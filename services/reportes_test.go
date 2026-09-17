package services

import (
	"testing"
	"time"

	"sistema-cuentas/models"
)

func TestGenerarChequesCirculacion(t *testing.T) {
	cuentas := []models.Cuenta{{ID: 1, Numero: "001", Nombre: "Operativa"}, {ID: 2, Numero: "002", Nombre: "Ahorro"}}
	cooperativas := []models.Cooperativa{{ID: 1, Nombre: "Coop Uno", NIT: "36029785"}}
	bancos := []models.Banco{{ID: 1, Nombre: "Banco Demo"}}

	corte := models.NuevaFecha(2026, time.September, 30)
	movimientos := []models.Movimiento{
		// Emitido hace 95 días: en circulación.
		{ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, NumeroDocumento: "1001", Beneficiario: "Ana",
			FechaOperacion: models.NuevaFecha(2026, time.June, 27), Estado: models.EstadoEgresoEmitido},
		// Cobrado a tiempo: fuera.
		{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 50, NumeroDocumento: "1002",
			FechaOperacion: models.NuevaFecha(2026, time.September, 5), Estado: models.EstadoEgresoCobrado,
			FechaCobro: models.NuevaFecha(2026, time.September, 10)},
		// Cobrado después del corte: en circulación al corte.
		{ID: 3, CuentaID: 2, Tipo: models.TipoEgreso, Monto: 25, NumeroDocumento: "2001",
			FechaOperacion: models.NuevaFecha(2026, time.September, 20), Estado: models.EstadoEgresoCobrado,
			FechaCobro: models.NuevaFecha(2026, time.October, 2)},
		// Anulado: no cuenta.
		{ID: 4, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 10, NumeroDocumento: "1003",
			FechaOperacion: models.NuevaFecha(2026, time.September, 1), Estado: models.EstadoEgresoAnulado},
	}

	r := GenerarChequesCirculacion(cuentas, cooperativas, bancos, movimientos, 0, corte)
	if r.Conteo != 2 {
		t.Fatalf("se esperaban 2 cheques en circulación, hay %d", r.Conteo)
	}
	if !Iguales(r.Monto, 125) {
		t.Fatalf("monto total esperado 125, obtenido %v", r.Monto)
	}
	// El cheque de 95 días debe ir primero (reporte ordenado por antigüedad).
	if r.Cheques[0].NumeroDocumento != "1001" || r.Cheques[0].DiasEnCirculacion != 95 {
		t.Fatalf("el cheque más antiguo debe encabezar: %+v", r.Cheques[0])
	}
	if r.Cheques[0].Rango != "90+" || r.Cheques[1].Rango != "0-29" {
		t.Fatalf("rangos de edad incorrectos: %s y %s", r.Cheques[0].Rango, r.Cheques[1].Rango)
	}
	if r.PorEdad["90+"].Conteo != 1 {
		t.Fatalf("debe haber 1 cheque de 90+ días: %+v", r.PorEdad)
	}
}

func TestGenerarChequesCirculacionFiltraPorCuenta(t *testing.T) {
	cuentas := []models.Cuenta{{ID: 1}, {ID: 2}}
	movimientos := []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 100, Estado: models.EstadoEgresoEmitido,
			FechaOperacion: models.NuevaFecha(2026, time.September, 1)},
		{ID: 2, CuentaID: 2, Tipo: models.TipoEgreso, Monto: 200, Estado: models.EstadoEgresoEmitido,
			FechaOperacion: models.NuevaFecha(2026, time.September, 1)},
	}
	corte := models.NuevaFecha(2026, time.September, 30)
	r := GenerarChequesCirculacion(cuentas, nil, nil, movimientos, 2, corte)
	if r.Conteo != 1 || r.Cheques[0].CuentaID != 2 {
		t.Fatalf("el filtro por cuenta falló: %+v", r.Cheques)
	}
}

func TestGenerarResumenMensual(t *testing.T) {
	cuentas := []models.Cuenta{{ID: 1, Numero: "001", Nombre: "Operativa", SaldoInicial: 1000}}
	cooperativas := []models.Cooperativa{{ID: 1, Nombre: "Coop", NIT: "36029785"}}
	bancos := []models.Banco{{ID: 1, Nombre: "Banco"}}
	movimientos := []models.Movimiento{
		// Del mes anterior: alimenta el saldo inicial del periodo.
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 200,
			FechaOperacion: models.NuevaFecha(2026, time.January, 15), Estado: models.EstadoIngresoActivo},
		// Febrero.
		{ID: 2, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 500,
			FechaOperacion: models.NuevaFecha(2026, time.February, 10), Estado: models.EstadoIngresoActivo},
		{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 300,
			FechaOperacion: models.NuevaFecha(2026, time.February, 20), Estado: models.EstadoEgresoEmitido},
	}

	r := GenerarResumenMensual(cuentas, cooperativas, bancos, movimientos, 2, 2026)
	if len(r.Filas) != 1 {
		t.Fatalf("se esperaba 1 fila, hay %d", len(r.Filas))
	}
	fila := r.Filas[0]
	if fila.SaldoInicialPeriodo != 1200 {
		t.Fatalf("saldo inicial de febrero = %v (esperado 1200)", fila.SaldoInicialPeriodo)
	}
	if fila.Depositos != 500 || fila.Cheques != 300 {
		t.Fatalf("movimientos del mes incorrectos: depósitos=%v cheques=%v", fila.Depositos, fila.Cheques)
	}
	if fila.SaldoFinal != 1400 {
		t.Fatalf("saldo final de febrero = %v (esperado 1400)", fila.SaldoFinal)
	}
	if r.Totales.Depositos != 500 || r.Totales.SaldoFinal != 1400 {
		t.Fatalf("totales incorrectos: %+v", r.Totales)
	}
}

func TestGenerarReporteAnual(t *testing.T) {
	cuenta := models.Cuenta{ID: 1, Numero: "001", Nombre: "Operativa", SaldoInicial: 1000}
	cooperativa := models.Cooperativa{ID: 1, Nombre: "Coop"}
	banco := models.Banco{ID: 1, Nombre: "Banco"}

	// Un depósito antes del año: solo desplaza el saldo inicial del año.
	previo := models.Movimiento{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 100,
		FechaOperacion: models.NuevaFecha(2025, time.December, 20), Estado: models.EstadoIngresoActivo}
	// Movimientos dentro del año.
	marzo := models.Movimiento{ID: 2, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 300,
		FechaOperacion: models.NuevaFecha(2026, time.March, 5), Estado: models.EstadoIngresoActivo}
	junio := models.Movimiento{ID: 3, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 150,
		FechaOperacion: models.NuevaFecha(2026, time.June, 10), Estado: models.EstadoEgresoEmitido}

	r := GenerarReporteAnual(cuenta, cooperativa, banco, []models.Movimiento{previo, junio, marzo}, 2026)
	if len(r.Filas) != 12 {
		t.Fatalf("se esperaban 12 meses, hay %d", len(r.Filas))
	}
	if r.SaldoInicialAnio != 1100 {
		t.Fatalf("saldo inicial del año = %v (esperado 1100)", r.SaldoInicialAnio)
	}
	if r.Filas[2].Depositos != 300 || r.Filas[5].Cheques != 150 {
		t.Fatalf("acumulación mensual incorrecta: %+v %+v", r.Filas[2], r.Filas[5])
	}
	if r.Filas[11].SaldoFinal != 1250 {
		t.Fatalf("saldo a diciembre = %v (esperado 1250)", r.Filas[11].SaldoFinal)
	}
	if r.TotalesDepositos != 300 || r.TotalesCheques != 150 {
		t.Fatalf("totales anuales incorrectos: %+v", r.TotalesDepositos)
	}
	// El saldo de los meses sin movimiento conserva el anterior.
	if r.Filas[1].SaldoFinal != 1100 {
		t.Fatalf("febrero no debe cambiar sin movimientos: %v", r.Filas[1].SaldoFinal)
	}
}

func TestRangoEdadCheque(t *testing.T) {
	casos := map[int]string{0: "0-29", 29: "0-29", 30: "30-59", 59: "30-59", 60: "60-89", 89: "60-89", 90: "90+", 400: "90+"}
	for dias, esperado := range casos {
		if got := RangoEdadCheque(dias); got != esperado {
			t.Fatalf("RangoEdadCheque(%d) = %s, esperado %s", dias, got, esperado)
		}
	}
}
