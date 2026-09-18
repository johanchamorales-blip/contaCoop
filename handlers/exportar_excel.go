package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sistema-cuentas/models"
	"sistema-cuentas/reportes"
	"sistema-cuentas/services"
)

// tipoXLSX es el Content-Type de los archivos creados por excelize.
const tipoXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// LibroBancosXLSX descarga el libro de bancos en el formato oficial (plantilla
// Excel) en lugar del resumen tipo factura. Reemplaza a window.print() y a la
// exportación CSV del libro.
func LibroBancosXLSX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	cuentaID := queryInt(r, "cuenta_id", 0)
	if cuentaID <= 0 {
		writeError(w, http.StatusBadRequest, "selecciona una cuenta para exportar")
		return
	}
	mes := queryInt(r, "mes", 0)
	anio := queryInt(r, "anio", 0)
	if mes < 0 || mes > 12 || anio < 0 {
		writeError(w, http.StatusBadRequest, "el periodo indicado no es válido")
		return
	}
	if mes > 0 && anio == 0 {
		writeError(w, http.StatusBadRequest, "indica el año del periodo")
		return
	}

	cuenta, cooperativa, banco, movimientos, err := contextoCuenta(cuentaID)
	if err != nil {
		responderError(w, err)
		return
	}
	usuario, _ := SesionDe(r)

	// Una llamada con todas las filas da la meta (cooperativa, banco, saldo
	// inicial) y el total exacto; luego se recorre por páginas para no cargar
	// todo el historial en memoria. La plantilla impresa admite 100
	// movimientos: si el período excede el límite se avisa para que el usuario
	// acote el rango antes de imprimir.
	completo := services.GenerarLibroBancos(cuenta, cooperativa, banco, movimientos, mes, anio, 1, 200)
	limite := 100
	total := completo.Meta.TotalRegistros

	movimientosExcel := make([]reportes.MovimientoBanco, 0, limite)
	escritas := 0
	pagina := 1
	for escritas < total {
		parcial := services.GenerarLibroBancos(cuenta, cooperativa, banco, movimientos, mes, anio, pagina, 200)
		if len(parcial.Filas) == 0 {
			break
		}
		for _, fila := range parcial.Filas {
			if len(movimientosExcel) == limite {
				writeError(w, http.StatusBadRequest, "el período tiene más de 100 movimientos: usa un mes concreto para imprimir")
				return
			}
			movimientosExcel = append(movimientosExcel, convertirFilaLibro(fila))
			escritas++
		}
		pagina++
	}

	datos := reportes.DatosCuenta{
		Empresa:      completo.Meta.Cooperativa,
		Banco:        completo.Meta.Banco,
		NumeroCuenta: completo.Meta.NumeroCuenta,
		Periodo:      periodoTexto(mes, anio),
		Moneda:       "Quetzales (GTQ)",
		SaldoInicial: completo.Meta.SaldoInicialPeriodo,
		Responsable:  usuario.Usuario,
	}

	filename := fmt.Sprintf("libro_bancos_%s", strings.ReplaceAll(cuenta.Numero, " ", "_"))
	if mes >= 1 && mes <= 12 && anio > 0 {
		filename += fmt.Sprintf("_%04d-%02d", anio, mes)
	}
	filename += ".xlsx"

	w.Header().Set("Content-Type", tipoXLSX)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if err := reportes.GenerarReporteLibroBancosWriter(datos, movimientosExcel, w); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo generar el archivo: "+err.Error())
		return
	}
}

// ConciliacionXLSX descarga una conciliación guardada en el formato oficial
// (plantilla Excel) en lugar del resumen tipo factura. Reemplaza a
// window.print() y a la exportación CSV de la conciliación.
func ConciliacionXLSX(w http.ResponseWriter, r *http.Request) {
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

	cuenta, cooperativa, banco, _, err := contextoCuenta(reporte.CuentaID)
	if err != nil {
		responderError(w, err)
		return
	}

	fechaConciliacion := fechaDDMMYYYY(reporte.FechaCorte)
	if fechaConciliacion == "" {
		fechaConciliacion = fechaDDMMYYYY(reporte.FechaSaldoEstadoCuenta)
	}

	datos := reportes.DatosCuenta{
		Empresa:           cooperativa.Nombre,
		Banco:             banco.Nombre,
		NumeroCuenta:      cuenta.Numero,
		Periodo:           periodoTexto(reporte.Mes, reporte.Anio),
		Moneda:            "Quetzales (GTQ)",
		Responsable:       reporte.Usuario,
		FechaConciliacion: fechaConciliacion,
	}

	filename := fmt.Sprintf("conciliacion_%04d-%02d_%d.xlsx", reporte.Anio, reporte.Mes, reporte.ID)

	w.Header().Set("Content-Type", tipoXLSX)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if err := reportes.GenerarReporteConciliacionWriter(
		datos, reporte.SaldoLibros, reporte.SaldoEstadoCuenta,
		partidasConciliacion(reporte), w,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo generar el archivo: "+err.Error())
		return
	}
}

// convertirFilaLibro adapta una fila del libro a la estructura que la plantilla
// espera. La descripción cae en la columna D; si no hay concepto se usa el
// beneficiario o remitente para que la fila nunca quede en blanco.
func convertirFilaLibro(fila services.LibroBancoFila) reportes.MovimientoBanco {
	tipo := "Cheque"
	if fila.Deposito > 0 {
		tipo = "Depósito"
	}
	descripcion := strings.TrimSpace(fila.Concepto)
	if descripcion == "" {
		switch {
		case fila.Beneficiario != "":
			descripcion = fila.Beneficiario
		case fila.Remitente != "":
			descripcion = fila.Remitente
		case fila.NumeroDocumento != "":
			descripcion = "Documento " + fila.NumeroDocumento
		}
	}
	return reportes.MovimientoBanco{
		Fecha:      fechaDDMMYYYY(fila.FechaOperacion),
		Documento:  fila.NumeroDocumento,
		Tipo:       tipo,
		Concepto:   descripcion,
		Ingreso:    fila.Deposito,
		Egreso:     fila.Cheque,
		Referencia: fila.Estado,
	}
}

// partidasConciliacion convierte las partidas guardadas de una conciliación en
// las filas que la plantilla suma en B13/C13. El signo de cada partida se
// acomoda para que los totales del archivo coincidan con los saldos ajustados
// que el usuario ya confirmó al guardar.
func partidasConciliacion(reporte *models.Conciliacion) []reportes.PartidaConciliatoria {
	partidas := make([]reportes.PartidaConciliatoria, 0, 5)
	if !services.EsCero(reporte.DepositosEnTransito) {
		partidas = append(partidas, reportes.PartidaConciliatoria{
			Fecha:       fechaDDMMYYYY(reporte.FechaDepositosEnTransito),
			Descripcion: "Depósitos realizados que el banco aún no ha acreditado",
			Tipo:        "Depósito en tránsito",
			AjusteBanco: reporte.DepositosEnTransito,
			Estado:      "Pendiente",
		})
	}
	if !services.EsCero(reporte.ChequesCirculacion) {
		partidas = append(partidas, reportes.PartidaConciliatoria{
			Fecha:       fechaDDMMYYYY(reporte.FechaCorte),
			Descripcion: "Cheques emitidos que aún no han sido cobrados",
			Tipo:        "Cheque en circulación",
			AjusteBanco: -reporte.ChequesCirculacion,
			Estado:      "Pendiente",
		})
	}
	if !services.EsCero(reporte.NotasCredito) {
		partidas = append(partidas, reportes.PartidaConciliatoria{
			Fecha:        fechaDDMMYYYY(reporte.FechaNotasCredito),
			Descripcion:  "Notas de crédito registradas por el banco",
			Tipo:         "Nota de crédito",
			AjusteLibros: reporte.NotasCredito,
			Estado:       "Conciliado",
		})
	}
	if !services.EsCero(reporte.NotasDebito) {
		partidas = append(partidas, reportes.PartidaConciliatoria{
			Fecha:        fechaDDMMYYYY(reporte.FechaNotasDebito),
			Descripcion:  "Notas de débito registradas por el banco",
			Tipo:         "Nota de débito",
			AjusteLibros: -reporte.NotasDebito,
			Estado:       "Conciliado",
		})
	}
	if !services.EsCero(reporte.Ajustes) || !services.EsCero(reporte.AjusteBanco) {
		descripcion := "Ajuste manual"
		if strings.TrimSpace(reporte.AjusteNombre) != "" {
			descripcion = "Ajuste manual (" + reporte.AjusteNombre + ")"
		}
		partidas = append(partidas, reportes.PartidaConciliatoria{
			Fecha:        fechaDDMMYYYY(reporte.FechaAjustes),
			Descripcion:  descripcion,
			Tipo:         "Ajuste",
			AjusteBanco:  reporte.AjusteBanco,
			AjusteLibros: reporte.Ajustes,
			Estado:       "Conciliado",
		})
	}
	return partidas
}

// fechaDDMMYYYY convierte una fecha modelo al formato que las plantillas usan.
func fechaDDMMYYYY(f models.Fecha) string {
	if f.EsVacia() {
		return ""
	}
	return f.Time.Format("02/01/2006")
}