package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

// ReporteChequesCirculacion lista los egresos que siguen sin cobrar a la fecha
// de corte, agrupados por antigüedad para encendender las alertas.
func ReporteChequesCirculacion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	corte, err := models.ParsearFecha(r.URL.Query().Get("corte"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de corte no es válida")
		return
	}

	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	movimientos, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarChequesCirculacion(cuentas, cooperativas, bancos, movimientos, cuentaID, corte)
	writeJSON(w, http.StatusOK, resultado)
}

func ReporteChequesCirculacionCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	corte, err := models.ParsearFecha(r.URL.Query().Get("corte"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de corte no es válida")
		return
	}

	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	movimientos, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarChequesCirculacion(cuentas, cooperativas, bancos, movimientos, cuentaID, corte)

	filename := "cheques_en_circulacion.csv"
	if cuentaID > 0 {
		if cuenta, ok := buscarCuenta(cuentas, cuentaID); ok {
			filename = "cheques_en_circulacion_" + strings.ReplaceAll(cuenta.Numero, " ", "_") + ".csv"
		}
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"REPORTE DE CHEQUES EN CIRCULACIÓN"})
	_ = cw.Write([]string{"Corte", resultado.Corte.String()})
	_ = cw.Write([]string{"Total", strconv.Itoa(resultado.Conteo) + " cheque(s)", formatearDecimal(resultado.Monto)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Rango de antigüedad", "Cheques", "Monto"})
	for _, rango := range []string{"0-29", "30-59", "60-89", "90+"} {
		entrada := resultado.PorEdad[rango]
		_ = cw.Write([]string{rango + " días", strconv.Itoa(entrada.Conteo), formatearDecimal(entrada.Monto)})
	}
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Fecha", "Cheque", "Beneficiario", "Concepto", "Cuenta", "Banco", "Cooperativa", "Monto", "Días", "Rango"})
	for _, c := range resultado.Cheques {
		_ = cw.Write([]string{
			c.FechaOperacion.String(),
			c.NumeroDocumento,
			c.Beneficiario,
			c.Concepto,
			c.NumeroCuenta,
			c.Banco,
			c.Cooperativa,
			formatearDecimal(c.Monto),
			strconv.Itoa(c.DiasEnCirculacion),
			c.Rango,
		})
	}
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Total", "", "", "", "", "", "", formatearDecimal(resultado.Monto), "", ""})
	cw.Flush()
}

// ReporteResumenMensual comprime el mes de todas las cuentas.
func ReporteResumenMensual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ahora := time.Now().UTC()
	mes := queryInt(r, "mes", int(ahora.Month()))
	anio := queryInt(r, "anio", ahora.Year())
	if mes < 1 || mes > 12 || anio <= 0 {
		writeError(w, http.StatusBadRequest, "el periodo indicado no es válido")
		return
	}

	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	movimientos, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarResumenMensual(cuentas, cooperativas, bancos, movimientos, mes, anio)
	writeJSON(w, http.StatusOK, resultado)
}

func ReporteResumenMensualCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ahora := time.Now().UTC()
	mes := queryInt(r, "mes", int(ahora.Month()))
	anio := queryInt(r, "anio", ahora.Year())
	if mes < 1 || mes > 12 || anio <= 0 {
		writeError(w, http.StatusBadRequest, "el periodo indicado no es válido")
		return
	}

	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	cooperativas, err := cooperativaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	bancos, err := bancoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	movimientos, err := movimientoStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarResumenMensual(cuentas, cooperativas, bancos, movimientos, mes, anio)

	filename := fmt.Sprintf("resumen_mensual_%04d-%02d.csv", anio, mes)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"RESUMEN MENSUAL POR CUENTA"})
	_ = cw.Write([]string{"Periodo", periodoTexto(mes, anio)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Cuenta", "Banco", "Cooperativa", "NIT", "Saldo inicial", "Depósitos", "Cheques", "Saldo final"})
	for _, fila := range resultado.Filas {
		_ = cw.Write([]string{
			fila.Numero + " " + fila.Nombre,
			fila.Banco,
			fila.Cooperativa,
			fila.NIT,
			formatearDecimal(fila.SaldoInicialPeriodo),
			formatearDecimal(fila.Depositos),
			formatearDecimal(fila.Cheques),
			formatearDecimal(fila.SaldoFinal),
		})
	}
	_ = cw.Write([]string{"TOTALES", "", "", "",
		"", formatearDecimal(resultado.Totales.Depositos),
		formatearDecimal(resultado.Totales.Cheques),
		formatearDecimal(resultado.Totales.SaldoFinal)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Elaboró", "Tesorero", "Vo. Bo.", "Presidente Comisión de Vigilancia"})
	cw.Flush()
}

// ReporteAnual pinta mes a mes una cuenta durante el año.
func ReporteAnual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta para el reporte anual")
		return
	}
	anio := queryInt(r, "anio", time.Now().UTC().Year())
	if anio <= 0 {
		writeError(w, http.StatusBadRequest, "el año no es válido")
		return
	}

	cuenta, cooperativa, banco, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarReporteAnual(cuenta, cooperativa, banco, movimientos, anio)
	writeJSON(w, http.StatusOK, resultado)
}

func ReporteAnualCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta para el reporte anual")
		return
	}
	anio := queryInt(r, "anio", time.Now().UTC().Year())
	if anio <= 0 {
		writeError(w, http.StatusBadRequest, "el año no es válido")
		return
	}

	cuenta, cooperativa, banco, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}

	resultado := services.GenerarReporteAnual(cuenta, cooperativa, banco, movimientos, anio)

	filename := fmt.Sprintf("reporte_anual_%s_%d.csv", strings.ReplaceAll(cuenta.Numero, " ", "_"), anio)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"REPORTE ANUAL"})
	_ = cw.Write([]string{"Cuenta", resultado.Numero + " " + resultado.Nombre})
	_ = cw.Write([]string{"Banco", resultado.Banco})
	_ = cw.Write([]string{"Cooperativa", resultado.Cooperativa})
	_ = cw.Write([]string{"Año", strconv.Itoa(resultado.Anio)})
	_ = cw.Write([]string{"Saldo inicial del año", formatearDecimal(resultado.SaldoInicialAnio)})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Mes", "Depósitos", "Cheques", "Saldo final"})
	for _, fila := range resultado.Filas {
		_ = cw.Write([]string{
			nombresMes[fila.Mes],
			formatearDecimal(fila.Depositos),
			formatearDecimal(fila.Cheques),
			formatearDecimal(fila.SaldoFinal),
		})
	}
	_ = cw.Write([]string{
		"TOTALES",
		formatearDecimal(resultado.TotalesDepositos),
		formatearDecimal(resultado.TotalesCheques),
		formatearDecimal(resultado.SaldoFinal),
	})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Elaboró", "Tesorero", "Vo. Bo.", "Presidente Comisión de Vigilancia"})
	cw.Flush()
}
