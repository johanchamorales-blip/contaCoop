package handlers

import (
	"encoding/csv"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func Movimientos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listarMovimientos(w, r)
	case http.MethodPost:
		crearMovimiento(w, r)
	case http.MethodDelete:
		anularMovimiento(w, r)
	default:
		methodNotAllowed(w)
	}
}

func listarMovimientos(w http.ResponseWriter, r *http.Request) {
	items, err := movimientosFiltrados(r)
	if err != nil {
		responderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// movimientosFiltrados aplica los filtros comunes de la lista y del CSV:
// cuenta, tipo, estado y rango de fechas de operación.
func movimientosFiltrados(r *http.Request) ([]models.Movimiento, error) {
	items, err := movimientoStore.Read()
	if err != nil {
		return nil, err
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("cuenta_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errParametros("cuenta_id inválido")
		}
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.CuentaID == id {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}
	if tipo := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("tipo"))); tipo != "" {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.Tipo == tipo {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}
	if estado := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("estado"))); estado != "" {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if m.Estado == estado {
				filtrados = append(filtrados, m)
			}
		}
		items = filtrados
	}

	desde, err := models.ParsearFecha(r.URL.Query().Get("desde"))
	if err != nil {
		return nil, errParametros("la fecha desde no es válida")
	}
	hasta, err := models.ParsearFecha(r.URL.Query().Get("hasta"))
	if err != nil {
		return nil, errParametros("la fecha hasta no es válida")
	}
	if !desde.EsVacia() || !hasta.EsVacia() {
		filtrados := make([]models.Movimiento, 0)
		for _, m := range items {
			if !desde.EsVacia() && m.FechaOperacion.Antes(desde) {
				continue
			}
			if !hasta.EsVacia() && m.FechaOperacion.Despues(hasta) {
				continue
			}
			filtrados = append(filtrados, m)
		}
		items = filtrados
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].FechaOperacion.Time.Equal(items[j].FechaOperacion.Time) {
			return items[i].ID > items[j].ID
		}
		return items[i].FechaOperacion.Despues(items[j].FechaOperacion)
	})

	if limite := queryInt(r, "limite", 0); limite > 0 && len(items) > limite {
		items = items[:limite]
	}
	return items, nil
}

// MovimientosCSV exporta el listado filtrado de movimientos, igual a como se
// ve en pantalla, para trabajar fuera del sistema.
func MovimientosCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, err := movimientosFiltrados(r)
	if err != nil {
		responderError(w, err)
		return
	}
	cuentas, err := cuentaStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	filename := "movimientos.csv"
	raw := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("tipo")))
	if raw == models.TipoIngreso || raw == models.TipoEgreso {
		filename = strings.ToLower(raw) + "s.csv"
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"REPORTE DE MOVIMIENTOS"})
	_ = cw.Write([]string{"Cuenta", cuentaDeMovimientosLabel(cuentas, r)})
	_ = cw.Write([]string{"Filtros", strings.Join(filtrosMovimientosTexto(r), " · ")})
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Fecha operación", "Fecha registro", "Tipo", "Cheque/doc", "Remitente", "Beneficiario", "Solicitante", "Concepto", "Monto", "Estado", "Cuenta"})

	var totalDepositos, totalCheques float64
	for _, m := range items {
		cuenta, _ := buscarCuenta(cuentas, m.CuentaID)
		_ = cw.Write([]string{
			m.FechaOperacion.String(),
			m.FechaRegistro.Format("2006-01-02 15:04"),
			m.Tipo,
			m.NumeroDocumento,
			m.Remitente,
			m.Beneficiario,
			m.Solicitante,
			m.Concepto,
			formatearDecimal(m.Monto),
			m.Estado,
			cuenta.Numero,
		})
		if !m.Afecta() {
			continue
		}
		switch m.Tipo {
		case models.TipoIngreso:
			totalDepositos = services.Suma(totalDepositos, m.Monto)
		case models.TipoEgreso:
			totalCheques = services.Suma(totalCheques, m.Monto)
		}
	}
	_ = cw.Write([]string{"TOTALES", "", "", "", "", "", "", "",
		formatearDecimal(services.Suma(totalDepositos, totalCheques)), "", ""})
	_ = cw.Write([]string{"", "Depósitos", formatearDecimal(totalDepositos), "Cheques", formatearDecimal(totalCheques)})
	cw.Flush()
}

func cuentaDeMovimientosLabel(cuentas []models.Cuenta, r *http.Request) string {
	raw := strings.TrimSpace(r.URL.Query().Get("cuenta_id"))
	if raw == "" {
		return "Todas las cuentas"
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return "Todas las cuentas"
	}
	if cuenta, ok := buscarCuenta(cuentas, id); ok {
		return cuenta.Numero + " " + cuenta.Nombre
	}
	return "Todas las cuentas"
}

func filtrosMovimientosTexto(r *http.Request) []string {
	partes := make([]string, 0, 4)
	agregar := func(clave string) {
		if valor := strings.TrimSpace(r.URL.Query().Get(clave)); valor != "" {
			partes = append(partes, clave+" = "+valor)
		}
	}
	agregar("tipo")
	agregar("estado")
	agregar("desde")
	agregar("hasta")
	if len(partes) == 0 {
		return []string{"sin filtros"}
	}
	return partes
}

func crearMovimiento(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		CuentaID           int     `json:"cuenta_id"`
		Tipo               string  `json:"tipo"`
		FechaOperacion     string  `json:"fecha_operacion"`
		FechaEmisionCheque string  `json:"fecha_emision_cheque"`
		FechaDeposito      string  `json:"fecha_deposito"`
		NumeroDocumento    string  `json:"numero_documento"`
		Remitente          string  `json:"remitente"`
		Beneficiario       string  `json:"beneficiario"`
		Solicitante        string  `json:"solicitante"`
		Concepto           string  `json:"concepto"`
		Monto              float64 `json:"monto"`
	}
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}

	fechaOperacion, err := models.ParsearFecha(entrada.FechaOperacion)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de operación no es válida")
		return
	}
	fechaEmision, err := models.ParsearFecha(entrada.FechaEmisionCheque)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de emisión no es válida")
		return
	}
	fechaDeposito, err := models.ParsearFecha(entrada.FechaDeposito)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha del depósito no es válida")
		return
	}

	usuario, _ := SesionDe(r)
	movimiento := models.Movimiento{
		CuentaID:           entrada.CuentaID,
		Tipo:               entrada.Tipo,
		FechaOperacion:     fechaOperacion,
		FechaEmisionCheque: fechaEmision,
		FechaDeposito:      fechaDeposito,
		NumeroDocumento:    entrada.NumeroDocumento,
		Remitente:          entrada.Remitente,
		Beneficiario:       entrada.Beneficiario,
		Solicitante:        entrada.Solicitante,
		Concepto:           entrada.Concepto,
		Monto:              entrada.Monto,
		Usuario:            usuario.Usuario,
	}

	errTx := enTransaccion(func() error {
		cuentas, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}

		resultado, err := services.CrearMovimiento(movimiento, cuentas, movimientos, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		movimientos = append(movimientos, resultado.Movimiento)
		services.RecalcularSaldos(cuentas, movimientos)

		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		if err := cuentaStore.Write(cuentas); err != nil {
			return err
		}

		cuenta, _ := buscarCuenta(cuentas, resultado.Movimiento.CuentaID)
		auditar(r, "REGISTRÓ "+resultado.Movimiento.Tipo,
			services.DocumentoMovimiento(resultado.Movimiento, cuenta.Numero),
			services.FormatearQuetzales(resultado.Movimiento.Monto)+" · "+resultado.Movimiento.Concepto)

		writeJSON(w, http.StatusCreated, map[string]any{
			"movimiento": resultado.Movimiento,
			"avisos":     resultado.Avisos,
			"saldo":      cuenta.SaldoActual,
		})
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

// CobrarCheque marca un egreso como pagado por el banco. Hasta ese momento el
// cheque se considera en circulación en la conciliación.
func CobrarCheque(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var entrada struct {
		ID         int    `json:"id"`
		FechaCobro string `json:"fecha_cobro"`
	}
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	fechaCobro, err := models.ParsearFecha(entrada.FechaCobro)
	if err != nil {
		writeError(w, http.StatusBadRequest, "la fecha de cobro no es válida")
		return
	}

	errTx := enTransaccion(func() error {
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}
		movimiento, err := services.MarcarChequeCobrado(movimientos, entrada.ID, fechaCobro, time.Now().UTC())
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "el movimiento no existe")
				return nil
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}
		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		cuentas, _ := cuentaStore.Read()
		cuenta, _ := buscarCuenta(cuentas, movimiento.CuentaID)
		auditar(r, "REGISTRÓ COBRO DE CHEQUE",
			services.DocumentoMovimiento(movimiento, cuenta.Numero),
			"Cobrado el "+movimiento.FechaCobro.String())
		writeJSON(w, http.StatusOK, movimiento)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

func anularMovimiento(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("id")))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "indica el movimiento a anular")
		return
	}
	motivo := strings.TrimSpace(r.URL.Query().Get("motivo"))
	usuario, _ := SesionDe(r)

	errTx := enTransaccion(func() error {
		movimientos, err := movimientoStore.Read()
		if err != nil {
			return err
		}
		movimiento, err := services.AnularMovimiento(movimientos, id, motivo, usuario.Usuario, time.Now().UTC())
		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "el movimiento no existe")
				return nil
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		cuentas, err := cuentaStore.Read()
		if err != nil {
			return err
		}
		services.RecalcularSaldos(cuentas, movimientos)

		if err := movimientoStore.Write(movimientos); err != nil {
			return err
		}
		if err := cuentaStore.Write(cuentas); err != nil {
			return err
		}

		cuenta, _ := buscarCuenta(cuentas, movimiento.CuentaID)
		auditar(r, "ANULÓ "+movimiento.Tipo,
			services.DocumentoMovimiento(movimiento, cuenta.Numero),
			motivo)
		writeJSON(w, http.StatusOK, movimiento)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}
