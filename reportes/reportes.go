// Package reportes exporta a Excel los reportes bancarios (libro de bancos y
// conciliación bancaria) a partir de plantillas con fórmulas predefinidas.
//
// Las plantillas base se leen de la carpeta plantillas/ y el resultado se
// guarda en la ruta indicada (normalmente generados/). Se escriben únicamente
// las celdas de datos; las celdas con fórmulas (saldos acumulados, totales y
// estados) se respetan tal como existen en la plantilla.
package reportes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

const (
	// CarpetaPlantillas contiene las plantillas base usadas como punto de partida.
	CarpetaPlantillas = "plantillas"

	// Hojas dentro de cada plantilla.
	hojaLibroBancos  = "Libro de Bancos"
	hojaResumen      = "Resumen"
	hojaConciliacion = "Conciliación"

	// Rango de movimientos dentro de la hoja "Libro de Bancos".
	filaInicialMovimientos = 13
	filaFinalMovimientos   = 112

	// Rango de partidas conciliatorias dentro de la hoja "Conciliación".
	filaInicialPartidas = 20
	filaFinalPartidas   = 69
)

// PlantillaLibroBancos y PlantillaConciliacion son las rutas de las plantillas
// base. Son variables para permitir sobrescribirlas en entornos de prueba.
var (
	PlantillaLibroBancos  = filepath.Join(CarpetaPlantillas, "Libro_de_Bancos.xlsx")
	PlantillaConciliacion = filepath.Join(CarpetaPlantillas, "Conciliacion_Bancaria.xlsx")
)

// Columnas de la hoja "Libro de Bancos" (A=1, B=2, ...).
const (
	libroColFecha = 1 + iota // A: Fecha
	libroColDocumento        // B: No. Documento
	libroColTipo             // C: Tipo
	libroColConcepto         // D: Descripción / Concepto
	libroColIngreso          // E: Ingreso
	libroColEgreso           // F: Egreso
	libroColSaldo            // G: Saldo (fórmula, no se escribe)
	libroColReferencia       // H: Referencia
)

// Columnas de la hoja "Conciliación" (A=1, B=2, ...).
const (
	concColFecha = 1 + iota // A: Fecha
	concColReferencia       // B: Referencia
	concColDescripcion      // C: Descripción
	concColTipo             // D: Tipo de diferencia
	concColAjusteBanco      // E: Ajuste al banco
	concColAjusteLibros     // F: Ajuste a libros
	concColEstado           // G: Estado
	concColObservacion      // H: Observación
	concColImporte          // I: Importe (fórmula, no se escribe)
)

// MovimientoBanco representa una fila del libro de bancos.
type MovimientoBanco struct {
	Fecha      string  `json:"fecha"`       // DD/MM/YYYY
	Documento  string  `json:"documento"`   // No. de documento
	Tipo       string  `json:"tipo"`        // Depósito, Cheque, Transferencia, ...
	Concepto   string  `json:"concepto"`    // Descripción / concepto
	Ingreso    float64 `json:"ingreso"`     // Monto de ingreso
	Egreso     float64 `json:"egreso"`      // Monto de egreso
	Referencia string  `json:"referencia"`  // Referencia interna
}

// PartidaConciliatoria representa una fila de la conciliación bancaria.
type PartidaConciliatoria struct {
	Fecha        string  `json:"fecha"`          // DD/MM/YYYY
	Referencia   string  `json:"referencia"`     // Referencia de la partida
	Descripcion  string  `json:"descripcion"`    // Descripción / concepto
	Tipo         string  `json:"tipo"`           // Cheque en circulación, Depósito en tránsito, ...
	AjusteBanco  float64 `json:"ajuste_banco"`   // Madurez sobre el saldo del banco
	AjusteLibros float64 `json:"ajuste_libros"`  // Ajuste sobre el saldo de libros
	Estado       string  `json:"estado"`         // Pendiente, Conciliado, ...
	Observacion  string  `json:"observacion"`    // Nota u observación
}

// DatosCuenta agrupa los metadatos de la cuenta bancaria reportada.
type DatosCuenta struct {
	Empresa           string  `json:"empresa"`            // Nombre de la empresa / cooperativa
	Banco             string  `json:"banco"`              // Nombre del banco
	NumeroCuenta      string  `json:"numero_cuenta"`      // Número de cuenta
	Periodo           string  `json:"periodo"`            // Período reportado
	Moneda            string  `json:"moneda"`             // Código o nombre de la moneda
	SaldoInicial      float64 `json:"saldo_inicial"`      // Saldo inicial del período (celda H5)
	Responsable       string  `json:"responsable"`        // Nombre de quien elabora
	FechaConciliacion string  `json:"fecha_conciliacion"` // Fecha de la conciliación (DD/MM/YYYY)
}

// GenerarReporteLibroBancos abre la plantilla Libro_de_Bancos.xlsx, escribe el
// encabezado y los movimientos en el rango 13-112 y guarda el resultado en
// rutaSalida. Las celdas de fórmula (columna G y totales A9, D9, G9) no se
// tocan: se respetan las fórmulas existentes en la plantilla.
func GenerarReporteLibroBancos(datos DatosCuenta, movimientos []MovimientoBanco, rutaSalida string) error {
	f, err := llenarLibroBancos(datos, movimientos)
	if err != nil {
		return err
	}
	defer f.Close()
	return guardar(f, rutaSalida)
}

// GenerarReporteLibroBancosWriter escribe el reporte del libro de bancos
// directamente en w, sin pasar por un archivo temporal. Es la variante usada
// por los handlers HTTP para descargar el archivo en una sola pasada.
func GenerarReporteLibroBancosWriter(datos DatosCuenta, movimientos []MovimientoBanco, w io.Writer) error {
	f, err := llenarLibroBancos(datos, movimientos)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteTo(w); err != nil {
		return fmt.Errorf("escribir libro de bancos: %w", err)
	}
	return nil
}

// llenarLibroBancos abre la plantilla del libro de bancos, escribe el
// encabezado y los movimientos en el rango 13-112 y devuelve el archivo
// listo para guardar o escribir. El llamador es responsable de cerrarlo.
func llenarLibroBancos(datos DatosCuenta, movimientos []MovimientoBanco) (*excelize.File, error) {
	f, err := excelize.OpenFile(PlantillaLibroBancos)
	if err != nil {
		return nil, fmt.Errorf("abrir plantilla %s: %w", PlantillaLibroBancos, err)
	}
	if idx, _ := f.GetSheetIndex(hojaLibroBancos); idx < 0 {
		f.Close()
		return nil, fmt.Errorf("la plantilla %s no contiene la hoja %q", PlantillaLibroBancos, hojaLibroBancos)
	}

	// Encabezado / metadatos de la cuenta.
	encabezado := []struct {
		celda string
		valor any
	}{
		{"B4", datos.Empresa},
		{"E4", datos.Banco},
		{"H4", datos.NumeroCuenta},
		{"B5", datos.Periodo},
		{"E5", datos.Moneda},
		{"H5", datos.SaldoInicial}, // Crítico: de aquí dependen las fórmulas de saldo.
		{"B6", datos.Responsable},
	}
	for _, c := range encabezado {
		if err := escribirCelda(f, hojaLibroBancos, c.celda, c.valor); err != nil {
			f.Close()
			return nil, fmt.Errorf("encabezado libro de bancos: %w", err)
		}
	}

	// Detalle de movimientos: fila 13 en adelante.
	for i := range movimientos {
		fila := filaInicialMovimientos + i
		if fila > filaFinalMovimientos {
			f.Close()
			return nil, fmt.Errorf(
				"demasiados movimientos: la plantilla admite %d (filas %d-%d)",
				filaFinalMovimientos-filaInicialMovimientos+1,
				filaInicialMovimientos, filaFinalMovimientos,
			)
		}
		m := movimientos[i]
		filaCelda := func(col int) string { return nombreCelda(col, fila) }
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColFecha), m.Fecha); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColDocumento), m.Documento); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColTipo), m.Tipo); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColConcepto), m.Concepto); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColIngreso), m.Ingreso); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColEgreso), m.Egreso); err != nil {
			f.Close()
			return nil, err
		}
		// Columna G = saldo acumulado: NO se escribe (fórmula de la plantilla).
		if err := escribirCelda(f, hojaLibroBancos, filaCelda(libroColReferencia), m.Referencia); err != nil {
			f.Close()
			return nil, err
		}
	}

	// Fila de total al final de la tabla, con el total de gastos (egresos) y de
	// ingresos. Se escribe como fórmula para que Excel recalcule los importes
	// si el usuario edita o agrega movimientos.
	if len(movimientos) > 0 {
		filaTotal := filaInicialMovimientos + len(movimientos)
		ultima := filaInicialMovimientos + len(movimientos) - 1
		if filaTotal <= filaFinalMovimientos {
			celda := func(col int) string { return nombreCelda(col, filaTotal) }
			rango := func(col int) string {
				return fmt.Sprintf("SUM(%s:%s)", nombreCelda(col, filaInicialMovimientos), nombreCelda(col, ultima))
			}
			if err := escribirCelda(f, hojaLibroBancos, celda(libroColTipo), "TOTALES"); err != nil {
				f.Close()
				return nil, err
			}
			if err := escribirCelda(f, hojaLibroBancos, celda(libroColConcepto), "Total de gastos"); err != nil {
				f.Close()
				return nil, err
			}
			if err := f.SetCellFormula(hojaLibroBancos, celda(libroColIngreso), rango(libroColIngreso)); err != nil {
				f.Close()
				return nil, fmt.Errorf("total de ingresos del libro de bancos: %w", err)
			}
			if err := f.SetCellFormula(hojaLibroBancos, celda(libroColEgreso), rango(libroColEgreso)); err != nil {
				f.Close()
				return nil, fmt.Errorf("total de gastos del libro de bancos: %w", err)
			}
		}
	}

	return f, nil
}

// GenerarReporteConciliacion abre la plantilla Conciliacion_Bancaria.xlsx,
// escribe el encabezado, los saldos iniciales (B12, C12) y las partidas
// conciliatorias en el rango 20-69, y guarda el resultado en rutaSalida. Las
// celdas de fórmula (B13, C13, B14, C14, F12, F13 y columna I) no se tocan.
func GenerarReporteConciliacion(
	datos DatosCuenta,
	saldoLibros, saldoBanco float64,
	partidas []PartidaConciliatoria,
	rutaSalida string,
) error {
	f, err := llenarConciliacion(datos, saldoLibros, saldoBanco, partidas)
	if err != nil {
		return err
	}
	defer f.Close()
	return guardar(f, rutaSalida)
}

// GenerarReporteConciliacionWriter escribe el reporte de conciliación
// directamente en w, sin pasar por un archivo temporal. Es la variante usada
// por los handlers HTTP para descargar el archivo en una sola pasada.
func GenerarReporteConciliacionWriter(
	datos DatosCuenta,
	saldoLibros, saldoBanco float64,
	partidas []PartidaConciliatoria,
	w io.Writer,
) error {
	f, err := llenarConciliacion(datos, saldoLibros, saldoBanco, partidas)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteTo(w); err != nil {
		return fmt.Errorf("escribir conciliación bancaria: %w", err)
	}
	return nil
}

// llenarConciliacion abre la plantilla de conciliación, escribe el encabezado,
// los saldos iniciales (B12, C12) y las partidas conciliatorias en el rango
// 20-69, y devuelve el archivo listo para guardar o escribir. El llamador es
// responsable de cerrarlo.
func llenarConciliacion(
	datos DatosCuenta,
	saldoLibros, saldoBanco float64,
	partidas []PartidaConciliatoria,
) (*excelize.File, error) {
	f, err := excelize.OpenFile(PlantillaConciliacion)
	if err != nil {
		return nil, fmt.Errorf("abrir plantilla %s: %w", PlantillaConciliacion, err)
	}
	if idx, _ := f.GetSheetIndex(hojaConciliacion); idx < 0 {
		f.Close()
		return nil, fmt.Errorf("la plantilla %s no contiene la hoja %q", PlantillaConciliacion, hojaConciliacion)
	}

	// Datos generales de la conciliación.
	encabezado := []struct {
		celda string
		valor any
	}{
		{"B5", datos.Empresa},
		{"B6", datos.Banco},
		{"B7", datos.NumeroCuenta},
		{"B8", datos.Periodo},
		{"F5", datos.FechaConciliacion},
		{"F6", datos.Responsable},
	}
	for _, c := range encabezado {
		if err := escribirCelda(f, hojaConciliacion, c.celda, c.valor); err != nil {
			f.Close()
			return nil, fmt.Errorf("encabezado conciliación: %w", err)
		}
	}

	// Saldos iniciales: de ellos dependen las fórmulas del resumen.
	if err := escribirCelda(f, hojaConciliacion, "B12", saldoLibros); err != nil {
		f.Close()
		return nil, err
	}
	if err := escribirCelda(f, hojaConciliacion, "C12", saldoBanco); err != nil {
		f.Close()
		return nil, err
	}

	// Detalle de partidas conciliatorias: fila 20 en adelante.
	for i := range partidas {
		fila := filaInicialPartidas + i
		if fila > filaFinalPartidas {
			f.Close()
			return nil, fmt.Errorf(
				"demasiadas partidas: la plantilla admite %d (filas %d-%d)",
				filaFinalPartidas-filaInicialPartidas+1,
				filaInicialPartidas, filaFinalPartidas,
			)
		}
		p := partidas[i]
		filaCelda := func(col int) string { return nombreCelda(col, fila) }
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColFecha), p.Fecha); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColReferencia), p.Referencia); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColDescripcion), p.Descripcion); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColTipo), p.Tipo); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColAjusteBanco), p.AjusteBanco); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColAjusteLibros), p.AjusteLibros); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColEstado), p.Estado); err != nil {
			f.Close()
			return nil, err
		}
		if err := escribirCelda(f, hojaConciliacion, filaCelda(concColObservacion), p.Observacion); err != nil {
			f.Close()
			return nil, err
		}
		// Columna I = importe absoluto: NO se escribe (fórmula de la plantilla).
	}

	// Fila de total al final de la tabla de partidas, con la suma de los
	// ajustes al banco y a libros. Va como fórmula para que Excel la recalcule.
	if len(partidas) > 0 {
		filaTotal := filaInicialPartidas + len(partidas)
		ultima := filaInicialPartidas + len(partidas) - 1
		if filaTotal <= filaFinalPartidas {
			celda := func(col int) string { return nombreCelda(col, filaTotal) }
			rango := func(col int) string {
				return fmt.Sprintf("SUM(%s:%s)", nombreCelda(col, filaInicialPartidas), nombreCelda(col, ultima))
			}
			if err := escribirCelda(f, hojaConciliacion, celda(concColTipo), "TOTALES"); err != nil {
				f.Close()
				return nil, err
			}
			if err := escribirCelda(f, hojaConciliacion, celda(concColDescripcion), "Total de gastos"); err != nil {
				f.Close()
				return nil, err
			}
			if err := f.SetCellFormula(hojaConciliacion, celda(concColAjusteBanco), rango(concColAjusteBanco)); err != nil {
				f.Close()
				return nil, fmt.Errorf("total de ajustes al banco: %w", err)
			}
			if err := f.SetCellFormula(hojaConciliacion, celda(concColAjusteLibros), rango(concColAjusteLibros)); err != nil {
				f.Close()
				return nil, fmt.Errorf("total de ajustes a libros: %w", err)
			}
		}
	}

	return f, nil
}

// AsegurarPlantillas garantiza que las plantillas base existan en
// CarpetaPlantillas. Si alguna falta, se genera una equivalente con las celdas,
// fórmulas y formatos numéricos que las funciones de exportación esperan.
func AsegurarPlantillas() error {
	if err := os.MkdirAll(CarpetaPlantillas, 0o755); err != nil {
		return fmt.Errorf("crear carpeta %s: %w", CarpetaPlantillas, err)
	}
	entradas := []struct {
		ruta string
		gen  func(string) error
	}{
		{PlantillaLibroBancos, crearPlantillaLibroBancos},
		{PlantillaConciliacion, crearPlantillaConciliacion},
	}
	for _, e := range entradas {
		if _, err := os.Stat(e.ruta); os.IsNotExist(err) {
			if err := e.gen(e.ruta); err != nil {
				return fmt.Errorf("generar plantilla %s: %w", e.ruta, err)
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// escribirCelda asigna un valor a una celda manteniendo el formato de la
// plantilla. Los montos deben pasar como float64 y el texto como string.
func escribirCelda(f *excelize.File, hoja, celda string, valor any) error {
	if err := f.SetCellValue(hoja, celda, valor); err != nil {
		return fmt.Errorf("celda %s!%s: %w", hoja, celda, err)
	}
	return nil
}

// nombreCelda convierte coordenadas 1-based (columna, fila) al nombre tipo "B4".
func nombreCelda(col, fila int) string {
	nombre, _ := excelize.CoordinatesToCellName(col, fila)
	return nombre
}

// guardar crea la carpeta de salida si hace falta y escribe el archivo.
func guardar(f *excelize.File, rutaSalida string) error {
	dir := filepath.Dir(rutaSalida)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("crear carpeta de salida %s: %w", dir, err)
		}
	}
	if err := f.SaveAs(rutaSalida); err != nil {
		return fmt.Errorf("guardar reporte %s: %w", rutaSalida, err)
	}
	return nil
}

// estiloMoney devuelve (creando si hace falta) el estilo de número #,##0.00.
func estiloMoney(f *excelize.File) int {
	estilo, err := f.NewStyle(&excelize.Style{NumFmt: 4})
	if err != nil {
		return 0
	}
	return estilo
}