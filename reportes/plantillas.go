package reportes

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// crearPlantillaLibroBancos genera, solo como respaldo, una plantilla
// equivalente a plantillas/Libro_de_Bancos.xlsx con las mismas celdas,
// fórmulas y formatos numéricos que las funciones de exportación esperan.
func crearPlantillaLibroBancos(ruta string) error {
	f := excelize.NewFile()
	defer f.Close()

	hoja := hojaLibroBancos
	if err := f.SetSheetName("Sheet1", hoja); err != nil {
		return err
	}
	if _, err := f.NewSheet(hojaResumen); err != nil {
		return err
	}
	f.SetActiveSheet(0)

	// Encabezado / metadatos.
	titulos := map[string]string{
		"A1": "LIBRO DE BANCOS",
		"A2": "Registro y control de movimientos de la cuenta bancaria",
		"A4": "Empresa", "D4": "Banco", "G4": "Número de cuenta",
		"A5": "Período", "D5": "Moneda", "G5": "Saldo inicial",
		"A6": "Responsable", "D6": "Última actualización", "G6": "Estado",
		"A8": "TOTAL INGRESOS", "D8": "TOTAL EGRESOS", "G8": "SALDO ACTUAL",
		"A12": "Fecha", "B12": "No. Documento", "C12": "Tipo",
		"D12": "Descripción", "E12": "Ingreso", "F12": "Egreso",
		"G12": "Saldo", "H12": "Referencia",
	}
	for celda, texto := range titulos {
		_ = f.SetCellValue(hoja, celda, texto)
	}

	// Fórmulas de totales (no se sobrescriben en la exportación).
	fórmulas := map[string]string{
		"A9": "SUM(E13:E112)",
		"D9": "SUM(F13:F112)",
		"G9": "H5+A9-D9",
	}
	for celda, formula := range fórmulas {
		if err := f.SetCellFormula(hoja, celda, formula); err != nil {
			return err
		}
	}

	// Fórmula de saldo acumulado en la columna G.
	for fila := filaInicialMovimientos; fila <= filaFinalMovimientos; fila++ {
		formula := fmt.Sprintf("IF(A%d=\"\",\"\",$H$5+E%d-F%d)", fila, fila, fila)
		if err := f.SetCellFormula(hoja, nombreCelda(libroColSaldo, fila), formula); err != nil {
			return err
		}
	}

	return aplicarFormatosYGuardar(f, ruta, func(estilo int) error {
		_ = setStyle(f, hoja, "H5", "H5", estilo)
		_ = setStyle(f, hoja, "G13", "G112", estilo)
		_ = setStyle(f, hoja, "E13", "F112", estilo)
		_ = setStyle(f, hoja, "A9", "A9", estilo)
		_ = setStyle(f, hoja, "D9", "D9", estilo)
		_ = setStyle(f, hoja, "G9", "G9", estilo)
		return nil
	})
}

// crearPlantillaConciliacion genera, solo como respaldo, una plantilla
// equivalente a plantillas/Conciliacion_Bancaria.xlsx.
func crearPlantillaConciliacion(ruta string) error {
	f := excelize.NewFile()
	defer f.Close()

	hoja := hojaConciliacion
	if err := f.SetSheetName("Sheet1", hoja); err != nil {
		return err
	}
	if _, err := f.NewSheet("Estado de Cuenta"); err != nil {
		return err
	}
	f.SetActiveSheet(0)

	// Datos generales y resumen de saldos.
	titulos := map[string]string{
		"A1": "CONCILIACIÓN BANCARIA",
		"A4": "DATOS GENERALES",
		"A5": "Empresa / Cuenta", "E5": "Fecha de conciliación",
		"A6": "Banco", "E6": "Responsable",
		"A7": "Número de cuenta", "A8": "Período",
		"A10": "RESUMEN DE LA CONCILIACIÓN",
		"A11": "CONCEPTO", "B11": "LIBROS", "C11": "BANCO", "E11": "RESULTADO",
		"A12": "Saldo inicial", "E12": "Diferencia",
		"A13": "Ajustes / partidas conciliatorias", "E13": "Estado",
		"A14": "Saldo ajustado", "E14": "Observación",
		"A17": "DETALLE DE PARTIDAS CONCILIATORIAS",
		"A19": "Fecha", "B19": "Referencia", "C19": "Descripción",
		"D19": "Tipo de diferencia", "E19": "Ajuste al banco",
		"F19": "Ajuste a libros", "G19": "Estado", "H19": "Observación",
		"I19": "Importe",
	}
	for celda, texto := range titulos {
		_ = f.SetCellValue(hoja, celda, texto)
	}

	// Fórmulas automáticas del resumen (no se sobrescriben en la exportación).
	fórmulas := map[string]string{
		"B13": "SUM(F20:F69)",
		"C13": "SUM(E20:E69)",
		"B14": "B12+B13",
		"C14": "C12+C13",
		"F12": "B14-C14",
		"F13": `IF(ABS(F12)<0.01,"CONCILIADA","PENDIENTE DE REVISIÓN")`,
	}
	for celda, formula := range fórmulas {
		if err := f.SetCellFormula(hoja, celda, formula); err != nil {
			return err
		}
	}

	// Fórmula de importe absoluto en la columna I.
	for fila := filaInicialPartidas; fila <= filaFinalPartidas; fila++ {
		formula := fmt.Sprintf("ABS(E%d)+ABS(F%d)", fila, fila)
		if err := f.SetCellFormula(hoja, nombreCelda(concColImporte, fila), formula); err != nil {
			return err
		}
	}

	return aplicarFormatosYGuardar(f, ruta, func(estilo int) error {
		_ = setStyle(f, hoja, "B12", "C12", estilo)
		_ = setStyle(f, hoja, "B13", "C14", estilo)
		_ = setStyle(f, hoja, "E20", "F69", estilo)
		_ = setStyle(f, hoja, "I20", "I69", estilo)
		_ = setStyle(f, hoja, "F12", "F13", estilo)
		return nil
	})
}

// aplicarFormatosYGuardar aplica los estilos numéricos y guarda el archivo.
func aplicarFormatosYGuardar(f *excelize.File, ruta string, aplicar func(int) error) error {
	estilo := estiloMoney(f)
	if estilo != 0 {
		if err := aplicar(estilo); err != nil {
			return err
		}
	}
	return f.SaveAs(ruta)
}

// setStyle aplica un estilo a un rango de celdas, ignorando errores de celdas
// inexistentes (los rangos se aplican a toda la plantilla).
func setStyle(f *excelize.File, hoja, inicio, fin string, estilo int) error {
	return f.SetCellStyle(hoja, inicio, fin, estilo)
}