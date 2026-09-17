package services

import (
	"sort"
	"time"

	"sistema-cuentas/models"
)

// ---------------------------------------------------------------------------
// Cheques en circulación
// ---------------------------------------------------------------------------

type ChequeCirculacionRow struct {
	MovimientoID      int          `json:"movimiento_id"`
	CuentaID          int          `json:"cuenta_id"`
	NumeroCuenta      string       `json:"numero_cuenta"`
	NombreCuenta      string       `json:"nombre_cuenta"`
	Banco             string       `json:"banco"`
	Cooperativa       string       `json:"cooperativa"`
	NIT               string       `json:"nit"`
	FechaOperacion    models.Fecha `json:"fecha_operacion"`
	NumeroDocumento   string       `json:"numero_documento"`
	Beneficiario      string       `json:"beneficiario"`
	Concepto          string       `json:"concepto"`
	Monto             float64      `json:"monto"`
	DiasEnCirculacion int          `json:"dias_en_circulacion"`
	Rango             string       `json:"rango"`
}

type ConteoChequesEdad struct {
	Conteo int     `json:"conteo"`
	Monto  float64 `json:"monto"`
}

type ResultadoChequesCirculacion struct {
	Corte   models.Fecha                 `json:"corte"`
	Cheques []ChequeCirculacionRow       `json:"cheques"`
	Conteo  int                          `json:"conteo"`
	Monto   float64                      `json:"monto_total"`
	PorEdad map[string]ConteoChequesEdad `json:"por_edad"`
}

func RangoEdadCheque(dias int) string {
	switch {
	case dias >= 90:
		return "90+"
	case dias >= 60:
		return "60-89"
	case dias >= 30:
		return "30-59"
	default:
		return "0-29"
	}
}

func DiasEntre(desde, hasta models.Fecha) int {
	if desde.EsVacia() || hasta.EsVacia() {
		return 0
	}
	return int(hasta.Time.Sub(desde.Time).Hours() / 24)
}

// GenerarChequesCirculacion arma el reporte de egresos que siguen pendientes
// de cobro al corte indicado. Con cuentaID=0 se incluyen todas las cuentas.
func GenerarChequesCirculacion(
	cuentas []models.Cuenta,
	cooperativas []models.Cooperativa,
	bancos []models.Banco,
	movimientos []models.Movimiento,
	cuentaID int,
	corte models.Fecha,
) ResultadoChequesCirculacion {

	if corte.EsVacia() {
		corte = models.FechaDesde(time.Now().UTC())
	}

	cheques := make([]ChequeCirculacionRow, 0)

	for _, m := range movimientos {
		if m.Tipo != models.TipoEgreso || !m.Afecta() || !EstaEnCirculacion(m, corte) {
			continue
		}
		if cuentaID > 0 && m.CuentaID != cuentaID {
			continue
		}

		cuenta, ok := buscarCuentaLocal(cuentas, m.CuentaID)
		if !ok {
			continue
		}
		cooperativa, _ := buscarCooperativaLocal(cooperativas, cuenta.CooperativaID)
		banco, _ := buscarBancoLocal(bancos, cuenta.BancoID)

		dias := DiasEntre(m.FechaOperacion, corte)

		cheques = append(cheques, ChequeCirculacionRow{
			MovimientoID:      m.ID,
			CuentaID:          cuenta.ID,
			NumeroCuenta:      cuenta.Numero,
			NombreCuenta:      cuenta.Nombre,
			Banco:             banco.Nombre,
			Cooperativa:       cooperativa.Nombre,
			NIT:               FormatearNIT(cooperativa.NIT),
			FechaOperacion:    m.FechaOperacion,
			NumeroDocumento:   m.NumeroDocumento,
			Beneficiario:      m.Beneficiario,
			Concepto:          m.Concepto,
			Monto:             Q(m.Monto),
			DiasEnCirculacion: dias,
			Rango:             RangoEdadCheque(dias),
		})
	}

	// Del más antiguo al más reciente: lo que lleva más tiempo sin cobrar se
	// ve primero.
	sort.SliceStable(cheques, func(i, j int) bool {
		if cheques[i].DiasEnCirculacion == cheques[j].DiasEnCirculacion {
			return cheques[i].NumeroDocumento < cheques[j].NumeroDocumento
		}
		return cheques[i].DiasEnCirculacion > cheques[j].DiasEnCirculacion
	})

	porEdad := map[string]ConteoChequesEdad{
		"0-29": {}, "30-59": {}, "60-89": {}, "90+": {},
	}
	var totalMonto float64
	for _, c := range cheques {
		entrada := porEdad[c.Rango]
		entrada.Conteo++
		entrada.Monto = Suma(entrada.Monto, c.Monto)
		porEdad[c.Rango] = entrada
		totalMonto = Suma(totalMonto, c.Monto)
	}

	return ResultadoChequesCirculacion{
		Corte:   corte,
		Cheques: cheques,
		Conteo:  len(cheques),
		Monto:   totalMonto,
		PorEdad: porEdad,
	}
}

// ---------------------------------------------------------------------------
// Resumen mensual por cuenta
// ---------------------------------------------------------------------------

type FilaResumenMensual struct {
	CuentaID            int     `json:"cuenta_id"`
	Numero              string  `json:"numero"`
	Nombre              string  `json:"nombre"`
	Banco               string  `json:"banco"`
	Cooperativa         string  `json:"cooperativa"`
	NIT                 string  `json:"nit"`
	SaldoInicialPeriodo float64 `json:"saldo_inicial_periodo"`
	Depositos           float64 `json:"depositos"`
	Cheques             float64 `json:"cheques"`
	SaldoFinal          float64 `json:"saldo_final"`
}

type TotalesResumenMensual struct {
	Depositos  float64 `json:"depositos"`
	Cheques    float64 `json:"cheques"`
	SaldoFinal float64 `json:"saldo_final"`
}

type ResultadoResumenMensual struct {
	Mes     int                   `json:"mes"`
	Anio    int                   `json:"anio"`
	Filas   []FilaResumenMensual  `json:"filas"`
	Totales TotalesResumenMensual `json:"totales"`
}

func GenerarResumenMensual(
	cuentas []models.Cuenta,
	cooperativas []models.Cooperativa,
	bancos []models.Banco,
	movimientos []models.Movimiento,
	mes int,
	anio int,
) ResultadoResumenMensual {

	inicioPeriodo := models.NuevaFecha(anio, time.Month(mes), 1)

	filas := make([]FilaResumenMensual, 0, len(cuentas))
	var totalDepositos, totalCheques, totalFinal float64

	for _, cuenta := range cuentas {
		cooperativa, _ := buscarCooperativaLocal(cooperativas, cuenta.CooperativaID)
		banco, _ := buscarBancoLocal(bancos, cuenta.BancoID)

		saldo := Q(cuenta.SaldoInicial)
		saldoInicialPeriodo := saldo

		var depositos, cheques float64

		ordenados := make([]models.Movimiento, 0)
		for _, m := range movimientos {
			if m.CuentaID == cuenta.ID && m.Afecta() {
				ordenados = append(ordenados, m)
			}
		}
		OrdenarCronologico(ordenados)

		for _, m := range ordenados {
			if !m.FechaOperacion.EnPeriodo(mes, anio) {
				if m.FechaOperacion.Antes(inicioPeriodo) {
					saldo = saldoDespuesMovimiento(saldo, m)
					saldoInicialPeriodo = saldo
				}
				continue
			}

			saldo = saldoDespuesMovimiento(saldo, m)
			switch m.Tipo {
			case models.TipoIngreso:
				depositos = Suma(depositos, m.Monto)
			case models.TipoEgreso:
				cheques = Suma(cheques, m.Monto)
			}
		}

		fila := FilaResumenMensual{
			CuentaID:            cuenta.ID,
			Numero:              cuenta.Numero,
			Nombre:              cuenta.Nombre,
			Banco:               banco.Nombre,
			Cooperativa:         cooperativa.Nombre,
			NIT:                 FormatearNIT(cooperativa.NIT),
			SaldoInicialPeriodo: saldoInicialPeriodo,
			Depositos:           depositos,
			Cheques:             cheques,
			SaldoFinal:          saldo,
		}
		filas = append(filas, fila)

		totalDepositos = Suma(totalDepositos, depositos)
		totalCheques = Suma(totalCheques, cheques)
		totalFinal = Suma(totalFinal, saldo)
	}

	sort.SliceStable(filas, func(i, j int) bool {
		if filas[i].Cooperativa == filas[j].Cooperativa {
			return filas[i].Numero < filas[j].Numero
		}
		return filas[i].Cooperativa < filas[j].Cooperativa
	})

	return ResultadoResumenMensual{
		Mes:   mes,
		Anio:  anio,
		Filas: filas,
		Totales: TotalesResumenMensual{
			Depositos:  totalDepositos,
			Cheques:    totalCheques,
			SaldoFinal: totalFinal,
		},
	}
}

// ---------------------------------------------------------------------------
// Reporte anual
// ---------------------------------------------------------------------------

type FilaReporteAnual struct {
	Mes        int     `json:"mes"`
	Depositos  float64 `json:"depositos"`
	Cheques    float64 `json:"cheques"`
	SaldoFinal float64 `json:"saldo_final"`
}

type ResultadoReporteAnual struct {
	Anio             int                `json:"anio"`
	CuentaID         int                `json:"cuenta_id"`
	Numero           string             `json:"numero"`
	Nombre           string             `json:"nombre"`
	Banco            string             `json:"banco"`
	Cooperativa      string             `json:"cooperativa"`
	SaldoInicialAnio float64            `json:"saldo_inicial_anio"`
	Filas            []FilaReporteAnual `json:"filas"`
	TotalesDepositos float64            `json:"totales_depositos"`
	TotalesCheques   float64            `json:"totales_cheques"`
	SaldoFinal       float64            `json:"saldo_final"`
}

func GenerarReporteAnual(
	cuenta models.Cuenta,
	cooperativa models.Cooperativa,
	banco models.Banco,
	movimientos []models.Movimiento,
	anio int,
) ResultadoReporteAnual {

	inicioAnio := models.NuevaFecha(anio, time.January, 1)

	ordenados := make([]models.Movimiento, 0)
	for _, m := range movimientos {
		if m.CuentaID == cuenta.ID && m.Afecta() {
			ordenados = append(ordenados, m)
		}
	}
	OrdenarCronologico(ordenados)

	saldo := Q(cuenta.SaldoInicial)
	saldoInicialAnio := saldo

	depositos := [13]float64{}
	cheques := [13]float64{}

	for _, m := range ordenados {
		if m.FechaOperacion.Antes(inicioAnio) {
			saldo = saldoDespuesMovimiento(saldo, m)
			saldoInicialAnio = saldo
			continue
		}
		if m.FechaOperacion.Time.Year() != anio {
			break
		}
		index := int(m.FechaOperacion.Time.Month())
		switch m.Tipo {
		case models.TipoIngreso:
			depositos[index] = Suma(depositos[index], m.Monto)
		case models.TipoEgreso:
			cheques[index] = Suma(cheques[index], m.Monto)
		}
	}

	saldoMes := saldoInicialAnio
	filas := make([]FilaReporteAnual, 0, 12)
	var totalDepositos, totalCheques float64

	for mes := 1; mes <= 12; mes++ {
		saldoMes = Suma(saldoMes, depositos[mes], -cheques[mes])
		filas = append(filas, FilaReporteAnual{
			Mes:        mes,
			Depositos:  depositos[mes],
			Cheques:    cheques[mes],
			SaldoFinal: saldoMes,
		})
		totalDepositos = Suma(totalDepositos, depositos[mes])
		totalCheques = Suma(totalCheques, cheques[mes])
	}

	return ResultadoReporteAnual{
		Anio:             anio,
		CuentaID:         cuenta.ID,
		Numero:           cuenta.Numero,
		Nombre:           cuenta.Nombre,
		Banco:            banco.Nombre,
		Cooperativa:      cooperativa.Nombre,
		SaldoInicialAnio: saldoInicialAnio,
		Filas:            filas,
		TotalesDepositos: totalDepositos,
		TotalesCheques:   totalCheques,
		SaldoFinal:       saldoMes,
	}
}
