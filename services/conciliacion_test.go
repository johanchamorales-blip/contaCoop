package services

import (
	"testing"
	"time"

	"sistema-cuentas/models"
)

func TestConciliacionSeparaLibrosDeBanco(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	movimientos := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.August, 3), Monto: 500, Estado: models.EstadoIngresoActivo},
		{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.August, 10), Monto: 200, Estado: models.EstadoEgresoEmitido, NumeroDocumento: "100"},
		{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, FechaOperacion: models.NuevaFecha(2026, time.August, 15), Monto: 50, Estado: models.EstadoEgresoCobrado, NumeroDocumento: "101", FechaCobro: models.NuevaFecha(2026, time.August, 20)},
		{ID: 4, CuentaID: 1, Tipo: models.TipoIngreso, FechaOperacion: models.NuevaFecha(2026, time.September, 2), Monto: 100, Estado: models.EstadoIngresoActivo},
	}

	// Libros al 31/08: 1000 + 500 - 200 - 50 = 1250.
	// Banco: 1450 (estado de cuenta) - 200 (cheque en circulación) = 1250.
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, models.Fecha{}, 1450, 0, 0, 0, 0, "", "", 0, "Guatemala, 31/08/2026", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana", time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.SaldoLibros != 1250 {
		t.Fatalf("saldo en libros = %v", r.Reporte.SaldoLibros)
	}
	if r.Reporte.ChequesCirculacion != 200 {
		t.Fatalf("cheques en circulación = %v", r.Reporte.ChequesCirculacion)
	}
	if r.Reporte.SaldoConciliado != 1250 || !r.Cuadrada {
		t.Fatalf("la conciliación debía cuadrar: %+v", r.Reporte)
	}
	if len(r.Cheques) != 1 || r.Cheques[0].NumeroDocumento != "100" {
		t.Fatalf("cheques en circulación incorrectos: %+v", r.Cheques)
	}
}

func TestConciliacionAplicaNotasDelLadoDeLosLibros(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	// Nota de débito de 25 (comisión que el banco ya cobró y los libros no tienen).
	r, err := GenerarConciliacion(cuenta, nil, 8, 2026, models.Fecha{}, 975, 0, 25, 0, 0, "", "", 0, "", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana", time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.SaldoLibrosAjustado != 975 {
		t.Fatalf("libros ajustados = %v", r.Reporte.SaldoLibrosAjustado)
	}
	if !r.Cuadrada {
		t.Fatalf("debía cuadrar: diferencia %v", r.Reporte.DiferenciaConLibros)
	}
}

func TestConciliacionCuentaChequeCobradoDespuesDelCierre(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 500}
	movimientos := []models.Movimiento{{
		ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, NumeroDocumento: "9",
		FechaOperacion: models.NuevaFecha(2026, time.August, 28),
		Estado:         models.EstadoEgresoCobrado,
		FechaCobro:     models.NuevaFecha(2026, time.September, 3),
	}}
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, models.Fecha{}, 500, 0, 0, 0, 0, "", "", 0, "", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.ChequesCirculacion != 100 {
		t.Fatalf("el cheque cobrado en septiembre seguía en circulación al 31/08: %v", r.Reporte.ChequesCirculacion)
	}
	if !r.Cuadrada {
		t.Fatalf("debía cuadrar: %+v", r.Reporte)
	}
}

func TestConciliacionRespetaFechaDeCortePersonalizada(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 500}
	movimientos := []models.Movimiento{
		// Cheque emitido el 10, cobrado el 20: al cortar el día 15 todavía
		// estaba en circulación aunque el mes siga abierto hasta el 31.
		{ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, NumeroDocumento: "1",
			FechaOperacion: models.NuevaFecha(2026, time.August, 10),
			Estado:         models.EstadoEgresoCobrado,
			FechaCobro:     models.NuevaFecha(2026, time.August, 20)},
	}

	corte := models.NuevaFecha(2026, time.August, 15)
	r, err := GenerarConciliacion(cuenta, movimientos, 8, 2026, corte, 400, 0, 0, 0, 0, "", "", 0, "", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana", time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.ChequesCirculacion != 100 {
		t.Fatalf("al cortar el 15, el cheque cobrado el 20 debía seguir en circulación: %v", r.Reporte.ChequesCirculacion)
	}
	if r.Reporte.FechaCorte.String() != "2026-08-15" {
		t.Fatalf("no guardó la fecha de corte: %v", r.Reporte.FechaCorte)
	}
}

func TestConciliacionRechazaFechaDeCorteFueraDelPeriodo(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 500}
	corte := models.NuevaFecha(2026, time.September, 2)
	_, err := GenerarConciliacion(cuenta, nil, 8, 2026, corte, 500, 0, 0, 0, 0, "", "", 0, "", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("debía rechazar una fecha de corte fuera del mes y año seleccionados")
	}
}

func TestConciliacionAjusteDebeAlBancoSuma(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	// Banco: 1500 + 200 de ajuste = 1700. Libros: 1000. Diferencia 700.
	r, err := GenerarConciliacion(cuenta, nil, 8, 2026, models.Fecha{}, 1500, 0, 0, 0, 0,
		"Depósito no contabilizado", AjusteBancoDebe, 200,
		"", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana",
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.AjusteBanco != 200 {
		t.Fatalf("ajuste al banco = %v", r.Reporte.AjusteBanco)
	}
	if r.Reporte.Ajustes != 0 {
		t.Fatalf("un ajuste al banco no debe tocar los libros: %v", r.Reporte.Ajustes)
	}
	if r.Reporte.SaldoConciliado != 1700 {
		t.Fatalf("saldo bancario ajustado = %v", r.Reporte.SaldoConciliado)
	}
	if r.Reporte.SaldoLibrosAjustado != 1000 {
		t.Fatalf("libros ajustados = %v", r.Reporte.SaldoLibrosAjustado)
	}
	if r.Reporte.AjusteNombre != "Depósito no contabilizado" || r.Reporte.AjusteTipo != AjusteBancoDebe {
		t.Fatalf("no guardó los datos del ajuste: %+v", r.Reporte)
	}
}

func TestConciliacionAjusteHaberALibrosResta(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	// Libros: 1000 - 150 de haber = 850. Banco: 850.
	r, err := GenerarConciliacion(cuenta, nil, 8, 2026, models.Fecha{}, 850, 0, 0, 0, 0,
		"Comisión no registrada", AjusteLibrosHaber, 150,
		"", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana",
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.Ajustes != -150 {
		t.Fatalf("ajuste a libros = %v", r.Reporte.Ajustes)
	}
	if r.Reporte.SaldoLibrosAjustado != 850 {
		t.Fatalf("libros ajustados = %v", r.Reporte.SaldoLibrosAjustado)
	}
	if !r.Cuadrada {
		t.Fatalf("debía cuadrar: diferencia %v", r.Reporte.DiferenciaConLibros)
	}
}

func TestConciliacionAjusteLegadoSinTipoSumaALibros(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	// Sin tipo, el valor firmado se aplica a libros como antes.
	r, err := GenerarConciliacion(cuenta, nil, 8, 2026, models.Fecha{}, 1350, 0, 0, 0, 350,
		"", "", 0, "", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana",
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.Reporte.Ajustes != 350 || r.Reporte.SaldoLibrosAjustado != 1350 {
		t.Fatalf("el ajuste heredado debía sumar a libros: %+v", r.Reporte)
	}
}

func TestConciliacionRechazaAjusteNegativo(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, SaldoInicial: 1000}
	_, err := GenerarConciliacion(cuenta, nil, 8, 2026, models.Fecha{}, 1000, 0, 0, 0, 0,
		"Error", AjusteLibrosDebe, -100,
		"", "", models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, "ana",
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("debía rechazar un ajuste con monto negativo")
	}
}
