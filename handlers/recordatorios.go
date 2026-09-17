package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"sistema-cuentas/models"
	"sistema-cuentas/services"
)

// Recordatorios administra las tareas pendientes del equipo. Se leen con
// cualquier sesión; se escriben con permiso de movimientos.
func Recordatorios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listarRecordatorios(w, r)
	case http.MethodPost:
		crearRecordatorio(w, r)
	case http.MethodPatch:
		actualizarRecordatorio(w, r)
	case http.MethodDelete:
		eliminarRecordatorio(w, r)
	default:
		methodNotAllowed(w)
	}
}

func listarRecordatorios(w http.ResponseWriter, r *http.Request) {
	items, err := recordatorioStore.Read()
	if err != nil {
		responderError(w, err)
		return
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("hecho")); raw != "" {
		hecho := raw == "true" || raw == "1"
		filtrados := make([]models.Recordatorio, 0)
		for _, item := range items {
			if item.Hecho == hecho {
				filtrados = append(filtrados, item)
			}
		}
		items = filtrados
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Hecho != items[j].Hecho {
			return !items[i].Hecho
		}
		if items[i].FechaLimite.EsVacia() || items[j].FechaLimite.EsVacia() {
			if items[i].FechaLimite.EsVacia() != items[j].FechaLimite.EsVacia() {
				return !items[i].FechaLimite.EsVacia()
			}
			return items[i].ID > items[j].ID
		}
		if items[i].FechaLimite.Time.Equal(items[j].FechaLimite.Time) {
			return items[i].ID > items[j].ID
		}
		return items[i].FechaLimite.Antes(items[j].FechaLimite)
	})

	writeJSON(w, http.StatusOK, items)
}

func crearRecordatorio(w http.ResponseWriter, r *http.Request) {
	var entrada models.Recordatorio
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}
	usuario, _ := SesionDe(r)

	errTx := enTransaccion(func() error {
		items, err := recordatorioStore.Read()
		if err != nil {
			return err
		}
		nuevo, err := services.CrearRecordatorio(items, entrada, usuario.Usuario, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}
		items = append(items, nuevo)
		if err := recordatorioStore.Write(items); err != nil {
			return err
		}
		auditar(r, "CREÓ RECORDATORIO", nuevo.Titulo, "Prioridad "+services.NombrePrioridad(nuevo.Prioridad))
		writeJSON(w, http.StatusCreated, nuevo)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

func actualizarRecordatorio(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		ID      int   `json:"id"`
		Hecho   *bool `json:"hecho"`
		Reabrir *bool `json:"reabrir"`
	}
	if err := readJSON(r, &entrada); err != nil {
		writeError(w, http.StatusBadRequest, "revisa los datos enviados")
		return
	}

	usuario, _ := SesionDe(r)
	errTx := enTransaccion(func() error {
		items, err := recordatorioStore.Read()
		if err != nil {
			return err
		}

		var modificado models.Recordatorio
		if entrada.Hecho != nil {
			if *entrada.Hecho {
				modificado, err = services.MarcarRecordatorioHecho(items, entrada.ID, usuario.Usuario, time.Now().UTC())
			} else {
				modificado, err = services.ReabrirRecordatorio(items, entrada.ID)
			}
		} else {
			writeError(w, http.StatusBadRequest, "indica qué cambio aplicar al recordatorio")
			return nil
		}

		if err != nil {
			if err == services.ErrNotFound {
				writeError(w, http.StatusNotFound, "el recordatorio no existe")
				return nil
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return nil
		}

		if err := recordatorioStore.Write(items); err != nil {
			return err
		}
		accion := "HABILITÓ RECORDATORIO"
		if modificado.Hecho {
			accion = "COMPLETÓ RECORDATORIO"
		}
		auditar(r, accion, modificado.Titulo, "")
		writeJSON(w, http.StatusOK, modificado)
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}

func eliminarRecordatorio(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("id")))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "indica el recordatorio a eliminar")
		return
	}

	errTx := enTransaccion(func() error {
		items, err := recordatorioStore.Read()
		if err != nil {
			return err
		}
		for i := range items {
			if items[i].ID != id {
				continue
			}
			eliminado := items[i]
			items = append(items[:i], items[i+1:]...)
			if err := recordatorioStore.Write(items); err != nil {
				return err
			}
			auditar(r, "ELIMINÓ RECORDATORIO", eliminado.Titulo, "")
			writeJSON(w, http.StatusOK, eliminado)
			return nil
		}
		writeError(w, http.StatusNotFound, "el recordatorio no existe")
		return nil
	})
	if errTx != nil {
		responderError(w, errTx)
	}
}
