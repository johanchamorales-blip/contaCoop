package services

import (
	"errors"
	"strings"
	"time"

	"sistema-cuentas/models"
)

var prioridadesValidas = map[string]string{
	models.PrioridadAlta:  "Alta",
	models.PrioridadMedia: "Media",
	models.PrioridadBaja:  "Baja",
}

func NombrePrioridad(prioridad string) string {
	if nombre, ok := prioridadesValidas[prioridad]; ok {
		return nombre
	}
	return prioridad
}

// CrearRecordatorio valida los datos y devuelve el recordatorio listo para
// guardar. La fecha límite no puede ser anterior a hoy.
func CrearRecordatorio(
	existentes []models.Recordatorio,
	r models.Recordatorio,
	usuario string,
	now time.Time,
) (models.Recordatorio, error) {

	r.Titulo = strings.TrimSpace(r.Titulo)
	r.Detalle = strings.TrimSpace(r.Detalle)
	r.Prioridad = strings.ToUpper(strings.TrimSpace(r.Prioridad))

	if r.Prioridad == "" {
		r.Prioridad = models.PrioridadMedia
	}
	if _, ok := prioridadesValidas[r.Prioridad]; !ok {
		return models.Recordatorio{}, errors.New("selecciona una prioridad válida")
	}

	if r.Titulo == "" {
		return models.Recordatorio{}, errors.New("el título del recordatorio es obligatorio")
	}

	hoy := models.FechaDesde(now)
	if !r.FechaLimite.EsVacia() && r.FechaLimite.Antes(hoy) {
		return models.Recordatorio{}, errors.New("la fecha límite no puede ser anterior a hoy")
	}

	maxID := 0
	for _, x := range existentes {
		if x.ID > maxID {
			maxID = x.ID
		}
	}

	r.ID = maxID + 1
	r.Hecho = false
	r.Usuario = strings.TrimSpace(usuario)
	r.FechaCreacion = now
	r.FechaRealizado = time.Time{}
	r.RealizadoPor = ""

	return r, nil
}

// MarcarRecordatorioHecho cierra un recordatorio pendiente.
func MarcarRecordatorioHecho(
	items []models.Recordatorio,
	id int,
	usuario string,
	now time.Time,
) (models.Recordatorio, error) {

	idx := indiceRecordatorio(items, id)
	if idx == -1 {
		return models.Recordatorio{}, ErrNotFound
	}

	r := items[idx]
	if r.Hecho {
		return models.Recordatorio{}, errors.New("ese recordatorio ya está completado")
	}

	r.Hecho = true
	r.FechaRealizado = now
	r.RealizadoPor = strings.TrimSpace(usuario)

	items[idx] = r
	return r, nil
}

// ReabrirRecordatorio devuelve un recordatorio completado a la lista de
// pendientes, por si se marcó por error o el pendiente volvió a surgir.
func ReabrirRecordatorio(items []models.Recordatorio, id int) (models.Recordatorio, error) {
	idx := indiceRecordatorio(items, id)
	if idx == -1 {
		return models.Recordatorio{}, ErrNotFound
	}

	r := items[idx]
	if !r.Hecho {
		return models.Recordatorio{}, errors.New("ese recordatorio todavía está pendiente")
	}

	r.Hecho = false
	r.FechaRealizado = time.Time{}
	r.RealizadoPor = ""

	items[idx] = r
	return r, nil
}

func indiceRecordatorio(items []models.Recordatorio, id int) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}
