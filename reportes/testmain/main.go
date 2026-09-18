// testmain es un programa independiente (package main) para probar la
// generación de reportes bancarios con los datos de ejemplo del documento
// "Automatización de Libros y Conciliaciones". No interfiere con el main()
// del servidor web, que vive en la raíz del repositorio.
package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"

	"sistema-cuentas/reportes"
)

const (
	hojaLibro  = "Libro de Bancos"
	hojaConc   = "Conciliación"
)

func main() {
	if err := reportes.AsegurarPlantillas(); err != nil {
		log.Fatalf("plantillas: %v", err)
	}

	datos := reportes.DatosCuenta{
		Empresa:           "Cooperativa de Desarrollo R.L.",
		Banco:             "Banco Industrial",
		NumeroCuenta:      "301-004589-2",
		Periodo:           "Septiembre 2026",
		Moneda:            "Quetzales (Q)",
		SaldoInicial:      50000.00,
		Responsable:       "Juan Pérez",
		FechaConciliacion: "30/09/2026",
	}

	movimientos := []reportes.MovimientoBanco{
		{Fecha: "01/09/2026", Documento: "DEP-001", Tipo: "Depósito", Concepto: "Aporte inicial de socios", Ingreso: 25000.00, Referencia: "Recibo #102"},
		{Fecha: "03/09/2026", Documento: "CH-1001", Tipo: "Cheque", Concepto: "Pago de alquiler de oficina", Egreso: 3500.00, Referencia: "Factura F-45"},
		{Fecha: "07/09/2026", Documento: "TRF-882", Tipo: "Transferencia", Concepto: "Cobro de servicio técnico", Ingreso: 8400.50, Referencia: "Factura F-108"},
		{Fecha: "12/09/2026", Documento: "CH-1002", Tipo: "Cheque", Concepto: "Compra de suministros de oficina", Egreso: 1250.75, Referencia: "Factura F-991"},
		{Fecha: "18/09/2026", Documento: "DEP-002", Tipo: "Depósito", Concepto: "Ventas al contado semana 2", Ingreso: 12300.00, Referencia: "Cierre #36"},
		{Fecha: "25/09/2026", Documento: "TRF-104", Tipo: "Transferencia", Concepto: "Pago de planilla de sueldos", Egreso: 14200.00, Referencia: "Planilla Sep"},
		{Fecha: "28/09/2026", Documento: "ND-044", Tipo: "Nota Débito", Concepto: "Manejo de cuenta mensual", Egreso: 150.00, Referencia: "Cargo Banco"},
	}

	const libroSalida = "generados/Libro_de_Bancos_Sep2026.xlsx"
	if err := reportes.GenerarReporteLibroBancos(datos, movimientos, libroSalida); err != nil {
		log.Fatalf("libro de bancos: %v", err)
	}
	fmt.Println("OK:", libroSalida)
	verificar(libroSalida, hojaLibro, []string{
		"B4", "E4", "H4", "B5", "E5", "H5", "B6",
		"A13", "B13", "C13", "D13", "E13", "F13", "H13",
		"A14", "B14", "C14", "D14", "E14", "F14", "H14",
		"A15", "B15", "C15", "D15", "E15", "F15", "H15",
		"A16", "B16", "C16", "D16", "E16", "F16", "H16",
		"A17", "B17", "C17", "D17", "E17", "F17", "H17",
		"A18", "B18", "C18", "D18", "E18", "F18", "H18",
		"A19", "B19", "C19", "D19", "E19", "F19", "H19",
	})

	partidas := []reportes.PartidaConciliatoria{
		{Fecha: "12/09/2026", Referencia: "CH-1002", Descripcion: "Cheque #1002 cobrado pero no liberado", Tipo: "Cheque en circulación", AjusteBanco: -1250.75, Estado: "Pendiente", Observacion: "Pendiente de cobro"},
		{Fecha: "18/09/2026", Referencia: "DEP-002", Descripcion: "Depósito en tránsito", Tipo: "Depósito en tránsito", AjusteBanco: 12300.00, Estado: "Pendiente", Observacion: "Fuera de horario"},
		{Fecha: "29/09/2026", Referencia: "NC-501", Descripcion: "Intereses ganados en cuenta", Tipo: "Nota de crédito", AjusteLibros: 500.00, Estado: "Pendiente", Observacion: "No en libros"},
		{Fecha: "30/09/2026", Referencia: "ND-099", Descripcion: "Comisión bancaria mensual", Tipo: "Nota de débito", AjusteLibros: -150.00, Estado: "Pendiente", Observacion: "Cargo automático"},
	}

	// Saldos que cuadran: B14 = 76599.75 + (500.00 - 150.00) = 76949.75;
	// C14 = 65900.50 + (12300.00 - 1250.75) = 76949.75. Diferencia F12 = 0.
	const conciliacionSalida = "generados/Conciliacion_Bancaria_Sep2026.xlsx"
	if err := reportes.GenerarReporteConciliacion(datos, 76599.75, 65900.50, partidas, conciliacionSalida); err != nil {
		log.Fatalf("conciliación: %v", err)
	}
	fmt.Println("OK:", conciliacionSalida)
	verificar(conciliacionSalida, hojaConc, []string{
		"B5", "B6", "B7", "B8", "F5", "F6", "B12", "C12",
		"A20", "B20", "C20", "D20", "E20", "F20", "G20", "H20",
		"A21", "B21", "C21", "D21", "E21", "F21", "G21", "H21",
		"A22", "B22", "C22", "D22", "E22", "F22", "G22", "H22",
		"A23", "B23", "C23", "D23", "E23", "F23", "G23", "H23",
	})
}

// verificar imprime el valor de celdas concretas del archivo generado para
// comprobar de un vistazo que quedaron exactamente como las pide el documento.
func verificar(archivo, hoja string, celdas []string) {
	f, err := excelize.OpenFile(archivo)
	if err != nil {
		log.Fatalf("abrir %s: %v", archivo, err)
	}
	defer f.Close()
	fmt.Printf("— %s (%s):\n", archivo, hoja)
	for _, celda := range celdas {
		valor, err := f.GetCellValue(hoja, celda)
		if err != nil {
			log.Fatalf("celda %s: %v", celda, err)
		}
		fmt.Printf("  %s = %s\n", celda, valor)
	}
}