package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sistema-cuentas/handlers"
	"sistema-cuentas/services"
)

func main() {
	if err := prepararDatos(); err != nil {
		log.Fatal(err)
	}

	// Si hay DATABASE_URL los datos se guardan en PostgreSQL, que sobrevive a
	// los reinicios del contenedor; si no, se usan los archivos de data/.
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		cerrar, err := services.UsarPostgres(dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer cerrar()
		if err := services.SembrarDesdeArchivos("data"); err != nil {
			log.Fatalf("sembrar datos iniciales: %v", err)
		}
		log.Println("Base de datos PostgreSQL conectada")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", handlers.Home)

	// Autenticación: lo único accesible sin sesión.
	mux.HandleFunc("/api/auth/sesion", handlers.Sesion)
	mux.HandleFunc("/api/auth/login", handlers.Login)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)
	mux.HandleFunc("/api/auth/registro", handlers.Registro)

	// Heartbeat para mantener activo el servicio en Render (plan gratuito)
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://contacoop-52c82.web.app")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// Catálogos y operación. El segundo argumento es el permiso exigido para
	// escribir; las lecturas quedan abiertas a cualquier sesión válida.
	mux.HandleFunc("/api/cooperativas", handlers.RequiereSesion("catalogos", handlers.Cooperativas))
	mux.HandleFunc("/api/bancos", handlers.RequiereSesion("catalogos", handlers.Bancos))
	mux.HandleFunc("/api/cuentas", handlers.RequiereSesion("catalogos", handlers.Cuentas))
	mux.HandleFunc("/api/movimientos", handlers.RequiereSesion("movimientos", handlers.Movimientos))
	mux.HandleFunc("/api/movimientos/cobrar", handlers.RequiereSesion("movimientos", handlers.CobrarCheque))
	mux.HandleFunc("/api/movimientos.csv", handlers.RequiereSesion("", handlers.MovimientosCSV))
	mux.HandleFunc("/api/libro-bancos", handlers.RequiereSesion("", handlers.LibroBancos))
	mux.HandleFunc("/api/libro-bancos.xlsx", handlers.RequiereSesion("", handlers.LibroBancosXLSX))
	mux.HandleFunc("/api/libro-bancos.csv", handlers.RequiereSesion("", handlers.LibroBancosCSV))
	mux.HandleFunc("/api/conciliaciones.csv", handlers.RequiereSesion("conciliacion", handlers.ConciliacionCSV))
	mux.HandleFunc("/api/conciliaciones.xlsx", handlers.RequiereSesion("conciliacion", handlers.ConciliacionXLSX))
	mux.HandleFunc("/api/conciliaciones/", handlers.RequiereSesion("conciliacion", handlers.ConciliacionGuardada))
	mux.HandleFunc("/api/conciliaciones", handlers.RequiereSesion("conciliacion", handlers.Conciliaciones))
	mux.HandleFunc("/api/conciliacion", handlers.RequiereSesion("", handlers.ConciliacionPreview))
	mux.HandleFunc("/api/busqueda", handlers.RequiereSesion("", handlers.Busqueda))
	mux.HandleFunc("/api/auditoria", handlers.RequiereSesion("", handlers.Auditoria))
	mux.HandleFunc("/api/resumen", handlers.RequiereSesion("", handlers.Resumen))
	mux.HandleFunc("/api/usuarios", handlers.RequiereSesion("usuarios", handlers.Usuarios))
	mux.HandleFunc("/api/recordatorios", handlers.RequiereSesion("movimientos", handlers.Recordatorios))

	mux.HandleFunc("/api/reportes/cheques-en-circulacion", handlers.RequiereSesion("", handlers.ReporteChequesCirculacion))
	mux.HandleFunc("/api/reportes/cheques-en-circulacion.csv", handlers.RequiereSesion("", handlers.ReporteChequesCirculacionCSV))
	mux.HandleFunc("/api/reportes/resumen-mensual", handlers.RequiereSesion("", handlers.ReporteResumenMensual))
	mux.HandleFunc("/api/reportes/resumen-mensual.csv", handlers.RequiereSesion("", handlers.ReporteResumenMensualCSV))
	mux.HandleFunc("/api/reportes/anual", handlers.RequiereSesion("", handlers.ReporteAnual))
	mux.HandleFunc("/api/reportes/anual.csv", handlers.RequiereSesion("", handlers.ReporteAnualCSV))

	archivos := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	mux.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sin Cache-Control el navegador puede seguir usando una versión
		// vieja de los archivos JS/CSS y de ahí falsos "revisa los datos".
		// "no-cache" obliga a revalidar en cada carga.
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Pragma", "no-cache")
		archivos.ServeHTTP(w, r)
	}))

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	if addr[0] != ':' {
		addr = ":" + addr
	}

	servidor := &http.Server{
		Addr:         addr,
		Handler:      registrarPeticiones(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Sistema de movimientos de cuentas en http://localhost%s\n", addr)
	log.Fatal(servidor.ListenAndServe())
}

func prepararDatos() error {
	if err := os.MkdirAll("data", 0o755); err != nil {
		return err
	}
	archivos := []string{
		"cooperativas.json", "bancos.json", "cuentas.json",
		"movimientos.json", "conciliaciones.json", "auditoria.json", "usuarios.json",
		"recordatorios.json",
	}
	for _, nombre := range archivos {
		ruta := filepath.Join("data", nombre)
		if _, err := os.Stat(ruta); os.IsNotExist(err) {
			if err := os.WriteFile(ruta, []byte("[]\n"), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func registrarPeticiones(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://contacoop-52c82.web.app")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		inicio := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/" && !filepath.HasPrefix(r.URL.Path, "/static") {
			log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(inicio).Round(time.Millisecond))
		}
	})
}
