// testmain es un programa independiente (package main) para probar la
// generación de reportes bancarios con datos de ejemplo. No interfiere con el
// main() del servidor web, que vive en la raíz del repositorio.
package main

import (
	"fmt"
	"log"

	"sistema-cuentas/reportes"
)

func main() {
	if err := reportes.AsegurarPlantillas(); err != nil {
		log.Fatalf("plantillas: %v", err)
	}

	datos := reportes.DatosCuenta{
		Empresa:           "Cooperativa de Ahorro y Crédito FÉNIX R.L.",
		Banco:             "Banco Agromercantil",
		NumeroCuenta:      "3152-463-8449-7",
		Periodo:           "Septiembre 2026",
		Moneda:            "Quetzales (GTQ)",
		SaldoInicial:      50000.00,
		Responsable:       "Juan Pérez",
		FechaConciliacion: "30/09/2026",
	}

	movimientos := []reportes.MovimientoBanco{
		{Fecha: "01/09/2026", Documento: "DEP-001", Tipo: "Depósito", Concepto: "Aportes de socios", Ingreso: 12500.00, Referencia: "045"},
		{Fecha: "03/09/2026", Documento: "CHQ-0012", Tipo: "Cheque", Concepto: "Pago a proveedor", Egreso: 3850.00, Referencia: "086"},
		{Fecha: "10/09/2026", Documento: "CHQ-0013", Tipo: "Cheque", Concepto: "Servicios administrativos", Egreso: 1200.50, Referencia: "091"},
		{Fecha: "15/09/2026", Documento: "TRF-002", Tipo: "Transferencia", Concepto: "Depósito por préstamo", Ingreso: 9000.00, Referencia: "102"},
		{Fecha: "25/09/2026", Documento: "CHQ-0014", Tipo: "Cheque", Concepto: "Honorarios contador", Egreso: 2500.00, Referencia: "118"},
	}

	const libroSalida = "generados/Libro_de_Bancos_Sep2026.xlsx"
	if err := reportes.GenerarReporteLibroBancos(datos, movimientos, libroSalida); err != nil {
		log.Fatalf("libro de bancos: %v", err)
	}
	fmt.Println("OK:", libroSalida)

	partidas := []reportes.PartidaConciliatoria{
		{Fecha: "02/09/2026", Referencia: "DEP-001", Descripcion: "Depósito aún no acreditado por el banco", Tipo: "Depósito en tránsito", AjusteBanco: 12500.00, Estado: "Pendiente", Observacion: "Acredita en 24-48 h"},
		{Fecha: "03/09/2026", Referencia: "CHQ-0012", Descripcion: "Cheque emitido aun sin cobrar", Tipo: "Cheque en circulación", AjusteBanco: -3850.00, Estado: "Pendiente", Observacion: "Puede presentarse hasta 03/10"},
		{Fecha: "21/09/2026", Referencia: "ND-0091", Descripcion: "Comisión bancaria por chequera", Tipo: "Nota de débito", AjusteLibros: -75.00, Estado: "Conciliado", Observacion: "Según estado de cuenta"},
		{Fecha: "10/09/2026", Referencia: "NC-003", Descripcion: "Intereses pagados por el banco", Tipo: "Nota de crédito", AjusteLibros: 120.40, Estado: "Conciliado", Observacion: ""},
	}

	// Saldos elegidos de modo que libros y banco cuadren y la fórmula F13
	// muestre "CONCILIADA". B14 = 48500 + (120.40 - 75.00) = 48545.40;
	// C14 = 39895.40 + (12500.00 - 3850.00) = 48545.40. Diferencia = 0.
	const conciliacionSalida = "generados/Conciliacion_Bancaria_Sep2026.xlsx"
	if err := reportes.GenerarReporteConciliacion(datos, 48500.00, 39895.40, partidas, conciliacionSalida); err != nil {
		log.Fatalf("conciliación: %v", err)
	}
	fmt.Println("OK:", conciliacionSalida)
}