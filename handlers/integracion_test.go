package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

// Suplanta las rutas de datos por un directorio temporal para no tocar los
// archivos reales del proyecto.
func dirigirStores(t *testing.T, dir string) {
	t.Helper()
	datos := filepath.Join(dir, "data")
	if err := os.MkdirAll(datos, 0o755); err != nil {
		t.Fatal(err)
	}

	cooperativaStore = services.JSONStore[models.Cooperativa]{Path: filepath.Join(datos, "cooperativas.json")}
	bancoStore = services.JSONStore[models.Banco]{Path: filepath.Join(datos, "bancos.json")}
	cuentaStore = services.JSONStore[models.Cuenta]{Path: filepath.Join(datos, "cuentas.json")}
	movimientoStore = services.JSONStore[models.Movimiento]{Path: filepath.Join(datos, "movimientos.json")}
	conciliacionStore = services.JSONStore[models.Conciliacion]{Path: filepath.Join(datos, "conciliaciones.json")}
	auditoriaStore = services.JSONStore[models.Auditoria]{Path: filepath.Join(datos, "auditoria.json")}
	usuarioStore = services.JSONStore[models.Usuario]{Path: filepath.Join(datos, "usuarios.json")}
	recordatorioStore = services.JSONStore[models.Recordatorio]{Path: filepath.Join(datos, "recordatorios.json")}

	catalogo := func(nombre string, lista any) {
		t.Helper()
		b, _ := json.Marshal(lista)
		if err := os.WriteFile(filepath.Join(datos, nombre), b, 0o644); err != nil {
			t.Fatalf("no se pudo escribir %s: %v", nombre, err)
		}
	}

	catalogo("cooperativas.json", []models.Cooperativa{{ID: 1, Nombre: "Coop Uno", NIT: "36029785", Direccion: "Sede central"}})
	catalogo("bancos.json", []models.Banco{{ID: 1, Nombre: "Banco Demo"}})
	catalogo("cuentas.json", []models.Cuenta{
		{ID: 1, CooperativaID: 1, BancoID: 1, Numero: "001", Nombre: "Operativa", SaldoInicial: 1000},
		{ID: 2, CooperativaID: 1, BancoID: 1, Numero: "002", Nombre: "Ahorro", SaldoInicial: 2000},
	})
	catalogo("movimientos.json", []models.Movimiento{
		{ID: 1, CuentaID: 1, Tipo: models.TipoIngreso, Monto: 500, Concepto: "Depósito enero", Remitente: "Socio",
			FechaOperacion: models.NuevaFecha(2026, 1, 10), FechaRegistro: models.NuevaFecha(2026, 1, 10).Time, Estado: models.EstadoIngresoActivo},
		{ID: 2, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 200, NumeroDocumento: "0001", Beneficiario: "Proveedor", Concepto: "Cheque viejo",
			FechaOperacion: models.NuevaFecha(2026, 4, 1), FechaRegistro: models.NuevaFecha(2026, 4, 1).Time, Estado: models.EstadoEgresoEmitido},
		{ID: 3, CuentaID: 2, Tipo: models.TipoIngreso, Monto: 300, Concepto: "Depósito mayo", Remitente: "Socio",
			FechaOperacion: models.NuevaFecha(2026, 5, 2), FechaRegistro: models.NuevaFecha(2026, 5, 2).Time, Estado: models.EstadoIngresoActivo},
		{ID: 4, CuentaID: 1, Tipo: models.TipoEgreso, Monto: 250, NumeroDocumento: "0002", Beneficiario: "Pago", Concepto: "Cheque reciente",
			FechaOperacion: models.NuevaFecha(2026, 9, 15), FechaRegistro: models.NuevaFecha(2026, 9, 15).Time, Estado: models.EstadoEgresoEmitido},
	})
	catalogo("conciliaciones.json", []models.Conciliacion{})
	catalogo("auditoria.json", []models.Auditoria{})
	catalogo("recordatorios.json", []models.Recordatorio{})
	catalogo("usuarios.json", []models.Usuario{})
}

func sesionCookie(t *testing.T) *http.Cookie {
	t.Helper()
	usuario := models.Usuario{ID: 1, Usuario: "Admin", Rol: "ADMINISTRADOR", Activo: true}
	prueba := httptest.NewRecorder()
	if err := abrirSesion(prueba, usuario); err != nil {
		t.Fatal(err)
	}
	if len(prueba.Result().Cookies()) == 0 {
		t.Fatal("no se creó la cookie de sesión")
	}
	return prueba.Result().Cookies()[0]
}

func peticion(t *testing.T, handler http.Handler, metodo, url string, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, url, strings.NewReader(cuerpo))
	if cuerpo != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(sesionCookie(t))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestUsuariosEliminar(t *testing.T) {
	dir := t.TempDir()
	dirigirStores(t, dir)
	m := http.NewServeMux()
	m.HandleFunc("/api/usuarios", RequiereSesion("usuarios", Usuarios))

	escribirUsuarios := func(usuarios []models.Usuario) {
		t.Helper()
		b, _ := json.Marshal(usuarios)
		if err := os.WriteFile(filepath.Join(dir, "data", "usuarios.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("elimina un usuario existente", func(t *testing.T) {
		escribirUsuarios([]models.Usuario{{ID: 1, Usuario: "Admin", Rol: models.RolAdministrador, Activo: true},
			{ID: 2, Usuario: "Pepe", Rol: models.RolContador, Activo: true}})

		rr := peticion(t, m, http.MethodDelete, "/api/usuarios?id=2", "")
		if rr.Code != http.StatusOK {
			t.Fatalf("eliminar: estado %d, cuerpo %s", rr.Code, rr.Body.String())
		}
		rr = peticion(t, m, http.MethodGet, "/api/usuarios", "")
		if !strings.Contains(rr.Body.String(), `"usuario":"Admin"`) || strings.Contains(rr.Body.String(), "Pepe") {
			t.Fatalf("tras eliminar debe quedar solo Admin: %s", rr.Body.String())
		}
	})

	t.Run("rechaza eliminar el propio acceso", func(t *testing.T) {
		escribirUsuarios([]models.Usuario{{ID: 1, Usuario: "Admin", Rol: models.RolAdministrador, Activo: true}})
		rr := peticion(t, m, http.MethodDelete, "/api/usuarios?id=1", "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("auto-borrado debería dar 400, dio %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("rechaza eliminar el último administrador", func(t *testing.T) {
		escribirUsuarios([]models.Usuario{{ID: 1, Usuario: "Admin", Rol: models.RolContador, Activo: true},
			{ID: 3, Usuario: "UnicaAdmin", Rol: models.RolAdministrador, Activo: true}})
		rr := peticion(t, m, http.MethodDelete, "/api/usuarios?id=3", "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("borrar el último administrador debería dar 400, dio %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("devuelve 404 si el usuario no existe", func(t *testing.T) {
		escribirUsuarios([]models.Usuario{{ID: 1, Usuario: "Admin", Rol: models.RolAdministrador, Activo: true}})
		rr := peticion(t, m, http.MethodDelete, "/api/usuarios?id=99", "")
		if rr.Code != http.StatusNotFound {
			t.Fatalf("borrar inexistente debería dar 404, dio %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestUsuariosDeshabilitar(t *testing.T) {
	dir := t.TempDir()
	dirigirStores(t, dir)
	m := http.NewServeMux()
	m.HandleFunc("/api/usuarios", RequiereSesion("usuarios", Usuarios))

	b, _ := json.Marshal([]models.Usuario{
		{ID: 1, Usuario: "Admin", Rol: models.RolAdministrador, Activo: true},
		{ID: 2, Usuario: "Pepe", Rol: models.RolContador, Activo: true},
	})
	if err := os.WriteFile(filepath.Join(dir, "data", "usuarios.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	rr := peticion(t, m, http.MethodPatch, "/api/usuarios", `{"id":2,"activo":false}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("deshabilitar: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"activo":false`) {
		t.Fatalf("no quedó inactivo: %s", rr.Body.String())
	}

	rr = peticion(t, m, http.MethodGet, "/api/usuarios", "")
	if !strings.Contains(rr.Body.String(), `"usuario":"Pepe"`) || !strings.Contains(rr.Body.String(), `"activo":false`) {
		t.Fatalf("el listado debía mostrar a Pepe inactivo: %s", rr.Body.String())
	}

	// No puede cambiar su propio acceso.
	rr = peticion(t, m, http.MethodPatch, "/api/usuarios", `{"id":1,"activo":false}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("modificar el propio acceso debía dar 400, dio %d: %s", rr.Code, rr.Body.String())
	}
}

func mux() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/recordatorios", RequiereSesion("movimientos", Recordatorios))
	m.HandleFunc("/api/reportes/cheques-en-circulacion", RequiereSesion("", ReporteChequesCirculacion))
	m.HandleFunc("/api/reportes/cheques-en-circulacion.csv", RequiereSesion("", ReporteChequesCirculacionCSV))
	m.HandleFunc("/api/reportes/resumen-mensual", RequiereSesion("", ReporteResumenMensual))
	m.HandleFunc("/api/reportes/resumen-mensual.csv", RequiereSesion("", ReporteResumenMensualCSV))
	m.HandleFunc("/api/reportes/anual", RequiereSesion("", ReporteAnual))
	m.HandleFunc("/api/reportes/anual.csv", RequiereSesion("", ReporteAnualCSV))
	m.HandleFunc("/api/movimientos.csv", RequiereSesion("", MovimientosCSV))
	m.HandleFunc("/api/movimientos", RequiereSesion("movimientos", Movimientos))
	m.HandleFunc("/api/resumen", RequiereSesion("", Resumen))
	return m
}

func TestCrearEgresoEIngresoPorHTTP(t *testing.T) {
	dir := t.TempDir()
	dirigirStores(t, dir)
	m := mux()

	// Mismo cuerpo que envía la vista de Egresos (incluye las fechas).
	rr := peticion(t, m, http.MethodPost, "/api/movimientos", `{
		"cuenta_id":1,
		"tipo":"EGRESO",
		"fecha_operacion":"2026-09-16",
		"fecha_emision_cheque":"2026-09-16",
		"fecha_deposito":"",
		"numero_documento":"0003",
		"beneficiario":"Proveedor SA",
		"solicitante":"Gerencia",
		"concepto":"Pago de planillas",
		"monto":350.50
	}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("crear egreso: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}
	var egreso struct {
		Movimiento models.Movimiento `json:"movimiento"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &egreso); err != nil {
		t.Fatal(err)
	}
	if egreso.Movimiento.Estado != models.EstadoEgresoEmitido {
		t.Fatalf("el egreso debía nacer emitido: %s", egreso.Movimiento.Estado)
	}

	// Mismo cuerpo que envía la vista de Ingresos (incluye la fecha del depósito).
	rr = peticion(t, m, http.MethodPost, "/api/movimientos", `{
		"cuenta_id":1,
		"tipo":"INGRESO",
		"fecha_operacion":"2026-09-16",
		"fecha_deposito":"2026-09-15",
		"numero_documento":"B-100",
		"remitente":"Socio Pérez",
		"concepto":"Depósito de planilla",
		"monto":400
	}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("crear ingreso: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}
	var ingreso struct {
		Movimiento models.Movimiento `json:"movimiento"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &ingreso); err != nil {
		t.Fatal(err)
	}
	if ingreso.Movimiento.FechaDeposito.String() != "2026-09-15" {
		t.Fatalf("no guardó la fecha del depósito: %+v", ingreso.Movimiento)
	}

	// Egreso con fecha de registro anterior a la emisión debe rechazarse, igual
	// que lo exige la regla del sistema.
	rr = peticion(t, m, http.MethodPost, "/api/movimientos", `{
		"cuenta_id":1,
		"tipo":"EGRESO",
		"fecha_operacion":"2026-09-15",
		"fecha_emision_cheque":"2026-09-16",
		"numero_documento":"0004",
		"beneficiario":"Proveedor SA",
		"solicitante":"Gerencia",
		"concepto":"Pago",
		"monto":50
	}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("egreso con registro anterior a la emisión debía rechazarse, dio %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	dirigirStores(t, dir)
	m := mux()

	casos := []struct {
		nombre   string
		url      string
		esperado string
	}{
		{nombre: "cheques JSON", url: "/api/reportes/cheques-en-circulacion?corte=2026-09-30", esperado: `"conteo":2`},
		{nombre: "cheques CSV", url: "/api/reportes/cheques-en-circulacion.csv?corte=2026-09-30", esperado: "REPORTE DE CHEQUES EN CIRCULACIÓN"},
		{nombre: "resumen mensual JSON", url: "/api/reportes/resumen-mensual?mes=1&anio=2026", esperado: `"depositos":500`},
		{nombre: "resumen mensual JSON febrero", url: "/api/reportes/resumen-mensual?mes=2&anio=2026", esperado: `"saldo_inicial_periodo":1500`},
		{nombre: "resumen mensual CSV", url: "/api/reportes/resumen-mensual.csv?mes=1&anio=2026", esperado: "RESUMEN MENSUAL POR CUENTA"},
		{nombre: "anual JSON", url: "/api/reportes/anual?cuenta_id=1&anio=2026", esperado: `"saldo_inicial_anio":1000`},
		{nombre: "anual CSV", url: "/api/reportes/anual.csv?cuenta_id=1&anio=2026", esperado: "REPORTE ANUAL"},
		{nombre: "movimientos CSV", url: "/api/movimientos.csv?cuenta_id=1", esperado: "REPORTE DE MOVIMIENTOS"},
		{nombre: "resumen del panel", url: "/api/resumen?mes=9&anio=2026", esperado: "CHEQUE_90"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rr := peticion(t, m, http.MethodGet, caso.url, "")
			if rr.Code != http.StatusOK {
				t.Fatalf("estado %d, cuerpo: %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), caso.esperado) {
				t.Fatalf("no aparece %q en: %s", caso.esperado, rr.Body.String())
			}
		})
	}
}

func TestRecordatoriosFlow(t *testing.T) {
	dir := t.TempDir()
	dirigirStores(t, dir)
	m := mux()

	rr := peticion(t, m, http.MethodPost, "/api/recordatorios",
		`{"titulo":"Reponer talonario","prioridad":"ALTA","fecha_limite":"2026-12-01","detalle":"Pedir al banco"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("crear recordatorio: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}
	var creado models.Recordatorio
	if err := json.Unmarshal(rr.Body.Bytes(), &creado); err != nil {
		t.Fatal(err)
	}

	rr = peticion(t, m, http.MethodGet, "/api/recordatorios?hecho=false", "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"titulo":"Reponer talonario"`) {
		t.Fatalf("listar pendientes: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}

	rr = peticion(t, m, http.MethodPatch, "/api/recordatorios",
		`{"id":`+strconv.Itoa(creado.ID)+`,"hecho":true}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("completar: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}

	rr = peticion(t, m, http.MethodGet, "/api/recordatorios?hecho=true", "")
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"hecho":true`) {
		t.Fatalf("listar completados: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}

	rr = peticion(t, m, http.MethodDelete, "/api/recordatorios?id="+strconv.Itoa(creado.ID), "")
	if rr.Code != http.StatusOK {
		t.Fatalf("eliminar: estado %d, cuerpo %s", rr.Code, rr.Body.String())
	}

	// Validación: prioridad inválida debe rechazarse.
	rr = peticion(t, m, http.MethodPost, "/api/recordatorios",
		`{"titulo":"Mal","prioridad":"EXTREMA"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("prioridad inválida debería dar 400, dio %d: %s", rr.Code, rr.Body.String())
	}
}
