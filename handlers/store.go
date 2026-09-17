package handlers

import (
	"path/filepath"
	"sync"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

func ruta(nombre string) string { return filepath.Join("data", nombre) }

// El Path es el respaldo local (tests y ejecución sin base de datos); la Clave
// es la llave en PostgreSQL cuando hay conexión configurada.
var (
	cooperativaStore  = services.JSONStore[models.Cooperativa]{Path: ruta("cooperativas.json"), Clave: "cooperativas"}
	bancoStore        = services.JSONStore[models.Banco]{Path: ruta("bancos.json"), Clave: "bancos"}
	cuentaStore       = services.JSONStore[models.Cuenta]{Path: ruta("cuentas.json"), Clave: "cuentas"}
	movimientoStore   = services.JSONStore[models.Movimiento]{Path: ruta("movimientos.json"), Clave: "movimientos"}
	conciliacionStore = services.JSONStore[models.Conciliacion]{Path: ruta("conciliaciones.json"), Clave: "conciliaciones"}
	auditoriaStore    = services.JSONStore[models.Auditoria]{Path: ruta("auditoria.json"), Clave: "auditoria"}
	usuarioStore      = services.JSONStore[models.Usuario]{Path: ruta("usuarios.json"), Clave: "usuarios"}
	recordatorioStore = services.JSONStore[models.Recordatorio]{Path: ruta("recordatorios.json"), Clave: "recordatorios"}
)

// transaccion serializa las operaciones que leen y escriben varios archivos.
// Sin este candado, dos registros simultáneos pueden leer la misma lista, a
// asignar el mismo ID y sobrescribirse entre sí.
var transaccion sync.Mutex

func enTransaccion(fn func() error) error {
	transaccion.Lock()
	defer transaccion.Unlock()
	return fn()
}
