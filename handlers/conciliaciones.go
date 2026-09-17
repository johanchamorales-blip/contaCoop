package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func Conciliaciones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listarConciliaciones(w, r)
	case http.MethodPost:
		guardarConciliacion(w, r)
	default:
		methodNotAllowed(w)
	}
}

func ConciliacionGuardada(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, err := strconv.Atoi(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/conciliaciones/"), "/"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "el identificador de conciliación no es válido")
		return
	}

	items, err := conciliacionStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	var reporte *models.Conciliacion
	for i := range items {
		if items[i].ID == id {
			reporte = &items[i]
			break
		}
	}
	if reporte == nil {
		writeError(w, http.StatusNotFound, "la conciliación no existe")
		return
	}

	cheques := reporte.ChequesEnCirculacionDetalle

	// Compatibilidad con conciliaciones antiguas que no guardaban el detalle
	// de cheques. En ese caso intentamos reconstruir únicamente esa parte,
	// dejando intactos los importes que realmente fueron guardados.
	if len(cheques) == 0 {
		if _, _, _, movimientos, err := contextoCuenta(reporte.CuentaID); err == nil {
			cierre := models.FechaDesde(
				models.NuevaFecha(reporte.Anio, time.Month(reporte.Mes), 1).
					Time.AddDate(0, 1, -1),
			)
			for _, m := range movimientos {
				if m.CuentaID == reporte.CuentaID && m.Afecta() && !m.FechaOperacion.Despues(cierre) && services.EstaEnCirculacion(m, cierre) {
					cheques = append(cheques, m)
				}
			}
			services.OrdenarCronologico(cheques)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reporte":                reporte,
		"cheques_en_circulacion": cheques,
		"cuadrada":               services.EsCero(reporte.DiferenciaConLibros),
	})
}

func ConciliacionCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("id")))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "indica la conciliación a exportar")
		return
	}

	items, err := conciliacionStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	var reporte *models.Conciliacion
	for i := range items {
		if items[i].ID == id {
			reporte = &items[i]
			break
		}
	}
	if reporte == nil {
		writeError(w, http.StatusNotFound, "la conciliación no existe")
		return
	}

	filename := fmt.Sprintf("conciliacion_%04d-%02d_%d.csv", reporte.Anio, reporte.Mes, reporte.ID)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"CONCILIACIÓN BANCARIA"})
	_ = cw.Write([]string{"Conciliación No.", strconv.Itoa(reporte.ID)})
	_ = cw.Write([]string{"Periodo", periodoTexto(reporte.Mes, reporte.Anio)})
	_ = cw.Write([]string{"Cuenta ID", strconv.Itoa(reporte.CuentaID)})
	_ = cw.Write([]string{"Fecha y lugar", reporte.FechaLugar})
	_ = cw.Write([]string{"Usuario", reporte.Usuario})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"CONCEPTO", "MONTO", "FECHA"})
	_ = cw.Write([]string{"Saldo según estado de cuenta", formatearDecimal(reporte.SaldoEstadoCuenta), reporte.FechaSaldoEstadoCuenta.String()})
	etiquetaAjuste := func(prefijo string) string {
		if reporte.AjusteNombre == "" {
			return prefijo
		}
		return prefijo + " (" + reporte.AjusteNombre + ")"
	}
	_ = cw.Write([]string{"Depósitos en tránsito", formatearDecimal(reporte.DepositosEnTransito), reporte.FechaDepositosEnTransito.String()})
	_ = cw.Write([]string{"Cheques en circulación", formatearDecimal(reporte.ChequesCirculacion), ""})
	if !services.EsCero(reporte.AjusteBanco) {
		_ = cw.Write([]string{etiquetaAjuste("Ajustes al banco"), formatearDecimal(reporte.AjusteBanco), reporte.FechaAjustes.String()})
	}
	_ = cw.Write([]string{"Saldo bancario ajustado", formatearDecimal(reporte.SaldoConciliado), ""})
	_ = cw.Write([]string{"Saldo en libros", formatearDecimal(reporte.SaldoLibros), ""})
	_ = cw.Write([]string{"Notas de crédito", formatearDecimal(reporte.NotasCredito), reporte.FechaNotasCredito.String()})
	_ = cw.Write([]string{"Notas de débito", formatearDecimal(reporte.NotasDebito), reporte.FechaNotasDebito.String()})
	_ = cw.Write([]string{etiquetaAjuste("Ajustes a libros"), formatearDecimal(reporte.Ajustes), reporte.FechaAjustes.String()})
	_ = cw.Write([]string{"Saldo en libros ajustado", formatearDecimal(reporte.SaldoLibrosAjustado), ""})
	_ = cw.Write([]string{"Diferencia", formatearDecimal(reporte.DiferenciaConLibros), ""})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"CHEQUES EN CIRCULACIÓN", "Fecha", "Cheque", "Beneficiario", "Concepto", "Monto"})
	for _, cheque := range reporte.ChequesEnCirculacionDetalle {
		_ = cw.Write([]string{
			"",
			cheque.FechaOperacion.String(),
			cheque.NumeroDocumento,
			cheque.Beneficiario,
			cheque.Concepto,
			formatearDecimal(cheque.Monto),
		})
	}
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Observaciones", reporte.Observaciones})
	_ = cw.Write([]string{"Estado", map[bool]string{true: "CUADRADA", false: "CON DIFERENCIA"}[services.EsCero(reporte.DiferenciaConLibros)]})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Elaboró", "Tesorero", "Vo. Bo.", "Presidente Comisión de Vigilancia"})
	cw.Flush()
}

func listarConciliaciones(w http.ResponseWriter, r *http.Request) {
	items, err := conciliacionStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("cuenta_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cuenta_id inválido")
			return
		}
		filtradas := make([]models.Conciliacion, 0)
		for _, item := range items {
			if item.CuentaID == id {
				filtradas = append(filtradas, item)
			}
		}
		items = filtradas
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Anio == items[j].Anio {
			return items[i].Mes > items[j].Mes
		}
		return items[i].Anio > items[j].Anio
	})
	writeJSON(w, http.StatusOK, items)
}

type entradaConciliacion struct {
	CuentaID                 int     `json:"cuenta_id"`
	Mes                      int     `json:"mes"`
	Anio                     int     `json:"anio"`
	FechaCorte               string  `json:"fecha_corte"`
	SaldoEstadoCuenta        float64 `json:"saldo_estado_cuenta"`
	DepositosEnTransito      float64 `json:"depositos_en_transito"`
	NotasDebito              float64 `json:"notas_debito"`
	NotasCredito             float64 `json:"notas_credito"`
	Ajustes                  float64 `json:"ajustes"`
	AjusteNombre             string  `json:"ajuste_nombre"`
	AjusteTipo               string  `json:"ajuste_tipo"`
	AjusteMonto              float64 `json:"ajuste_monto"`
	FechaSaldoEstadoCuenta   string  `json:"fecha_saldo_estado_cuenta"`
	FechaDepositosEnTransito string  `json:"fecha_depositos_en_transito"`
	FechaNotasDebito         string  `json:"fecha_notas_debito"`
	FechaNotasCredito        string  `json:"fecha_notas_credito"`
	FechaAjustes             string  `json:"fecha_ajustes"`
	FechaLugar               string  `json:"fecha_lugar"`
	Observaciones            string  `json:"observaciones"`
	ConfirmaCirculacion      bool    `json:"confirma_circulacion"`
}

// parsearFechasConciliacion convierte las fechas opcionales que acompañan a
// cada importe. Una fecha vacía es válida; una fecha mal escrita se rechaza.
func parsearFechasConciliacion(
	fechaSaldo, fechaTransito, fechaDebito, fechaCredito, fechaAjustes string,
) (models.Fecha, models.Fecha, models.Fecha, models.Fecha, models.Fecha, error) {
	parsear := func(clave, valor string) (models.Fecha, error) {
		fecha, err := models.ParsearFecha(valor)
		if err != nil {
			return models.Fecha{}, fmt.Errorf("la fecha de %s no es válida (usa AAAA-MM-DD)", clave)
		}
		return fecha, nil
	}
	saldo, err := parsear("saldo del estado de cuenta", fechaSaldo)
	if err != nil {
		return models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, err
	}
	transito, err := parsear("depósitos en tránsito", fechaTransito)
	if err != nil {
		return models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, err
	}
	debito, err := parsear("notas de débito", fechaDebito)
	if err != nil {
		return models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, err
	}
	credito, err := parsear("notas de crédito", fechaCredito)
	if err != nil {
		return models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, err
	}
	ajustes, err := parsear("ajustes", fechaAjustes)
	if err != nil {
		return models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, models.Fecha{}, err
	}
	return saldo, transito, debito, credito, ajustes, nil
}

func guardarConciliacion(w http.ResponseWriter, r *http.Request) {
	var input entradaConciliacion
	if err := readJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	fechaCorte, err := models.ParsearFecha(input.FechaCorte)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de corte no es válida")
		return
	}
	fechaSaldo, fechaTransito, fechaDebito, fechaCredito, fechaAjustes, err :=
		parsearFechasConciliacion(
			input.FechaSaldoEstadoCuenta,
			input.FechaDepositosEnTransito,
			input.FechaNotasDebito,
			input.FechaNotasCredito,
			input.FechaAjustes,
		)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	usuario, _ := SesionDe(r)

	errTx := enTransaccion(func() error {
		cuenta, _, _, movimientos, err := contextoCuenta(input.CuentaID)
		if err != nil {
			responderError(w, err)
			return nil
		}

		resultado, err := services.GenerarConciliacion(
			cuenta, movimientos, input.Mes, input.Anio, fechaCorte,
			input.SaldoEstadoCuenta, input.DepositosEnTransito,
			input.NotasDebito, input.NotasCredito, input.Ajustes,
			strings.TrimSpace(input.AjusteNombre), input.AjusteTipo, input.AjusteMonto,
			strings.TrimSpace(input.FechaLugar), strings.TrimSpace(input.Observaciones),
			fechaSaldo, fechaTransito, fechaDebito, fechaCredito, fechaAjustes,
			usuario.Usuario, time.Now().UTC(),
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		// Si queda dinero en circulación, el contador debe confirmar a
		// propósito que lo vio antes de guardar la conciliación.
		if len(resultado.Cheques) > 0 && !input.ConfirmaCirculacion {
			writeJSON(w, http.StatusConflict, map[string]any{
				"requiere_confirmacion": true,
				"mensaje":               "todavía hay cheques en circulación; revísalos antes de guardar",
				"resultado":             resultado,
			})
			return nil
		}

		items, err := conciliacionStore.Read()
		if err != nil {
			return err
		}
		for _, existente := range items {
			if existente.CuentaID == input.CuentaID && existente.Mes == input.Mes && existente.Anio == input.Anio {
				writeError(w, http.StatusConflict, "esa cuenta ya tiene una conciliación guardada para el periodo")
				return nil
			}
		}

		resultado.Reporte.ID = services.NumeroConciliacion(items)
		resultado.Reporte.ChequesEnCirculacionDetalle = append([]models.Movimiento(nil), resultado.Cheques...)
		items = append(items, resultado.Reporte)
		if err := conciliacionStore.Write(items); err != nil {
			return err
		}
		auditar(r, "GUARDÓ CONCILIACIÓN", "Cuenta "+cuenta.Numero, services.ResumenConciliacion(resultado.Reporte))
		writeJSON(w, http.StatusCreated, resultado)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

// ConciliacionPreview calcula la conciliación sin guardarla.
func ConciliacionPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta")
		return
	}
	mes := queryInt(r, "mes", 0)
	anio := queryInt(r, "anio", 0)

	fechaCorte, err := models.ParsearFecha(r.URL.Query().Get("fecha_corte"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de corte no es válida")
		return
	}

	valores := map[string]float64{}
	for _, clave := range []string{"saldo_estado_cuenta", "depositos_en_transito", "notas_debito", "notas_credito", "ajustes"} {
		valor, err := queryFloat(r, clave)
		if err != nil {
			writeError(w, http.StatusBadRequest, "el valor de "+strings.ReplaceAll(clave, "_", " ")+" no es un número")
			return
		}
		valores[clave] = valor
	}

	ajusteMonto, err := queryFloat(r, "ajuste_monto")
	if err != nil {
		writeError(w, http.StatusBadRequest, "el monto del ajuste no es un número")
		return
	}

	fechaSaldo, fechaTransito, fechaDebito, fechaCredito, fechaAjustes, err :=
		parsearFechasConciliacion(
			r.URL.Query().Get("fecha_saldo_estado_cuenta"),
			r.URL.Query().Get("fecha_depositos_en_transito"),
			r.URL.Query().Get("fecha_notas_debito"),
			r.URL.Query().Get("fecha_notas_credito"),
			r.URL.Query().Get("fecha_ajustes"),
		)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cuenta, _, _, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}
	usuario, _ := SesionDe(r)

	resultado, err := services.GenerarConciliacion(
		cuenta, movimientos, mes, anio, fechaCorte,
		valores["saldo_estado_cuenta"], valores["depositos_en_transito"],
		valores["notas_debito"], valores["notas_credito"], valores["ajustes"],
		strings.TrimSpace(r.URL.Query().Get("ajuste_nombre")),
		r.URL.Query().Get("ajuste_tipo"),
		ajusteMonto,
		strings.TrimSpace(r.URL.Query().Get("fecha_lugar")),
		strings.TrimSpace(r.URL.Query().Get("observaciones")),
		fechaSaldo, fechaTransito, fechaDebito, fechaCredito, fechaAjustes,
		usuario.Usuario, time.Now().UTC(),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resultado)
}
