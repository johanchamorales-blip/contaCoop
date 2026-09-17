package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// JSONStore guarda una lista de documentos como un solo documento JSON. En
// producción (Clave definida y conexión de base de datos activa) persiste en
// PostgreSQL; si se configura un Path se usa el archivo local, que es el modo
// de los tests y de ejecutar el sistema sin base de datos.
type JSONStore[T any] struct {
	Path  string
	Clave string
	Mu    sync.Mutex
}

var (
	// baseDatos se conecta en main con UsarPostgres. Hasta entonces nil.
	baseDatos *sql.DB
	// bloqueoBase protege el arranque de la conexión.
	bloqueoBase sync.Mutex
)

// UsarPostgres abre el pool de conexiones, crea la tabla de documentos y
// devuelve un cierre limpio. Desde ese momento los stores con Clave definida
// dejan de usar archivos.
func UsarPostgres(dsn string) (func() error, error) {
	bloqueoBase.Lock()
	defer bloqueoBase.Unlock()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir conexión a la base de datos: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("no se pudo conectar a la base de datos: %w", err)
	}

	const crearTabla = `
		CREATE TABLE IF NOT EXISTS json_store (
			clave          TEXT PRIMARY KEY,
			contenido      JSONB NOT NULL,
			actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
		)`
	if _, err := db.ExecContext(ctx, crearTabla); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("crear tabla json_store: %w", err)
	}

	baseDatos = db
	return func() error { return db.Close() }, nil
}

// SembrarDesdeArchivos importa el contenido de data/*.json solo cuando la base
// de datos aún no tiene esa clave. Así el primer arranque conserva los datos
// históricos del repositorio sin volver a pisarlos después.
func SembrarDesdeArchivos(carpeta string) error {
	if baseDatos == nil {
		return nil
	}
	claves := []string{
		"cooperativas", "bancos", "cuentas", "movimientos",
		"conciliaciones", "auditoria", "usuarios", "recordatorios",
	}
	for _, clave := range claves {
		ruta := filepath.Join(carpeta, clave+".json")
		b, err := os.ReadFile(ruta)
		if err != nil {
			continue
		}
		if !tieneFilas(b) {
			continue
		}
		_, err = baseDatos.Exec(
			`INSERT INTO json_store (clave, contenido, actualizado_en)
			 SELECT $1, $2::jsonb, now()
			 WHERE NOT EXISTS (SELECT 1 FROM json_store WHERE clave = $1)`,
			clave, string(b),
		)
		if err != nil {
			return fmt.Errorf("sembrar %s desde archivo: %w", clave, err)
		}
	}
	return nil
}

func tieneFilas(b []byte) bool {
	// Un documento con solo "[]" (o espacios) no vale la pena sembrarlo.
	var lista []json.RawMessage
	return json.Unmarshal(b, &lista) == nil && len(lista) > 0
}

func (s *JSONStore[T]) Read() ([]T, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.Clave != "" && baseDatos != nil {
		return s.leerDeBase()
	}
	return s.leerDeArchivo()
}

func (s *JSONStore[T]) Write(items []T) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.Clave != "" && baseDatos != nil {
		return s.escribirEnBase(items)
	}
	return s.escribirEnArchivo(items)
}

func (s *JSONStore[T]) leerDeBase() ([]T, error) {
	var contenido string
	err := baseDatos.QueryRow(
		`SELECT contenido::text FROM json_store WHERE clave = $1`, s.Clave,
	).Scan(&contenido)
	if errors.Is(err, sql.ErrNoRows) {
		return []T{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leer %s de la base de datos: %w", s.Clave, err)
	}
	var items []T
	if err := json.Unmarshal([]byte(contenido), &items); err != nil {
		return nil, fmt.Errorf("parsear %s de la base de datos: %w", s.Clave, err)
	}
	return items, nil
}

func (s *JSONStore[T]) escribirEnBase(items []T) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if _, err := baseDatos.Exec(
		`INSERT INTO json_store (clave, contenido, actualizado_en)
		 VALUES ($1, $2::jsonb, now())
		 ON CONFLICT (clave) DO UPDATE
		   SET contenido = EXCLUDED.contenido, actualizado_en = now()`,
		s.Clave, string(data),
	); err != nil {
		return fmt.Errorf("guardar %s en la base de datos: %w", s.Clave, err)
	}
	return nil
}

func (s *JSONStore[T]) leerDeArchivo() ([]T, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", s.Path, err)
	}
	var items []T
	if len(data) == 0 {
		return []T{}, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsear %s: %w", s.Path, err)
	}
	return items, nil
}

func (s *JSONStore[T]) escribirEnArchivo(items []T) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}