package services

import (
	"strings"
	"testing"
	"time"

	"sistema-cuentas/models"
)

func TestCrearRecordatorioValidaDatos(t *testing.T) {
	ahora := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)

	if _, err := CrearRecordatorio(nil, models.Recordatorio{
		Titulo: "  ", Prioridad: "MEDIA",
	}, "ana", ahora); err == nil || !strings.Contains(err.Error(), "título") {
		t.Fatalf("se esperaba error por título vacío: %v", err)
	}

	if _, err := CrearRecordatorio(nil, models.Recordatorio{
		Titulo: "Conciliar", Prioridad: "URGENTE",
	}, "ana", ahora); err == nil || !strings.Contains(err.Error(), "prioridad") {
		t.Fatalf("se esperaba error por prioridad inválida: %v", err)
	}

	if _, err := CrearRecordatorio(nil, models.Recordatorio{
		Titulo:      "Conciliar",
		Prioridad:   "ALTA",
		FechaLimite: models.NuevaFecha(2026, time.September, 1),
	}, "ana", ahora); err == nil || !strings.Contains(err.Error(), "anterior") {
		t.Fatalf("se esperaba error por fecha límite pasada: %v", err)
	}
}

func TestCrearRecordatorioAsignaPrioridadYID(t *testing.T) {
	ahora := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	existentes := []models.Recordatorio{{ID: 1}, {ID: 3}}

	r, err := CrearRecordatorio(existentes, models.Recordatorio{
		Titulo:      "Conciliar agosto",
		Detalle:     "Con el estado de cuenta",
		CuentaID:    5,
		FechaLimite: models.NuevaFecha(2026, time.October, 1),
	}, "ana", ahora)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != 4 {
		t.Fatalf("ID esperado 4, obtenido %d", r.ID)
	}
	if r.Prioridad != models.PrioridadMedia {
		t.Fatalf("prioridad por defecto debe ser media: %s", r.Prioridad)
	}
	if r.Hecho || r.Usuario != "ana" {
		t.Fatalf("el recordatorio debe nacer pendiente y con usuario: %+v", r)
	}
}

func TestRecordatorioVencido(t *testing.T) {
	hoy := models.NuevaFecha(2026, time.September, 16)

	pendiente := models.Recordatorio{FechaLimite: models.NuevaFecha(2026, time.September, 10)}
	if !pendiente.Vencido(hoy) {
		t.Fatal("un recordatorio pendiente con fecha pasada debe estar vencido")
	}

	hecho := models.Recordatorio{Hecho: true, FechaLimite: models.NuevaFecha(2026, time.September, 10)}
	if hecho.Vencido(hoy) {
		t.Fatal("un recordatorio completado nunca está vencido")
	}
}

func TestMarcarYReabrirRecordatorio(t *testing.T) {
	ahora := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	items := []models.Recordatorio{{ID: 1, Titulo: "Conciliar", Prioridad: "ALTA"}}

	r, err := MarcarRecordatorioHecho(items, 1, "ana", ahora)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Hecho || r.RealizadoPor != "ana" || r.FechaRealizado.IsZero() {
		t.Fatalf("cierre incorrecto: %+v", r)
	}

	if _, err := MarcarRecordatorioHecho(items, 1, "ben", ahora); err == nil {
		t.Fatal("no debe poder cerrarse dos veces el mismo recordatorio")
	}

	if _, err := MarcarRecordatorioHecho(items, 99, "ana", ahora); err != ErrNotFound {
		t.Fatalf("ID inexistente debe devolver ErrNotFound, obtenido %v", err)
	}

	reabierto, err := ReabrirRecordatorio(items, 1)
	if err != nil {
		t.Fatal(err)
	}
	if reabierto.Hecho || !reabierto.FechaRealizado.IsZero() {
		t.Fatalf("reabrir debe limpiar el cierre: %+v", reabierto)
	}
}
