import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar, opciones } from '../core/dom.js';
import { fecha, fechaHora, hoy } from '../core/formato.js';
import { exito, problema, pendiente } from '../core/notificaciones.js';

// Recordatorios: la lista de pendientes del equipo. Cada compromiso tiene un
// responsable implícito (quien lo crea), una prioridad y una fecha límite.
export class Recordatorios extends Vista {
  static titulo = 'Recordatorios';
  static glifo = '⚑';
  static descripcion = 'Pendientes y avisos del equipo, con fecha de cumplimiento.';

  constructor(contexto) {
    super(contexto);
    this.pendientes = [];
    this.completados = [];
  }

  plantilla() {
    const ahora = new Date();
    return `
      ${this.encabezado('Recordatorios', 'Tareas y compromisos con fecha límite.')}
      ${this.estado.puede('movimientos') ? `
      <div class="tarjeta">
        <h2>Nuevo recordatorio</h2>
        <form class="formulario" data-formulario="recordatorio" style="margin-top:14px">
          <div class="par">
            <label>Título<input name="titulo" maxlength="120" required placeholder="Ej.: Solicitar reposición del talonario"></label>
            <label>Fecha límite<input name="fecha_limite" type="date" min="${hoy()}"></label>
          </div>
          <div class="par">
            <label>Prioridad<select name="prioridad">
              <option value="ALTA">Alta</option>
              <option value="MEDIA" selected>Media</option>
              <option value="BAJA">Baja</option>
            </select></label>
            <label>Cuenta (opcional)
              <select name="cuenta_id">
                ${opciones(this.estado.cuentas, { valor: (c) => c.id, texto: (c) => `${c.numero} · ${c.nombre}`, vacio: 'Cualquiera' })}
              </select>
            </label>
          </div>
          <label class="ancho-total">Detalle<textarea name="detalle" rows="2" placeholder="Contexto para el responsable"></textarea></label>
          <div class="acciones"><button type="submit">Guardar recordatorio</button></div>
        </form>
      </div>` : ''}

      <div class="tarjeta">
        <h2>Pendientes</h2>
        <div class="lista" data-zona="pendientes" style="margin-top:14px"></div>
      </div>

      <div class="tarjeta">
        <h2>Completados</h2>
        <div class="lista" data-zona="completados" style="margin-top:14px"></div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="recordatorio"]', (form) => this.crear(form));
    this.alHacerClic('[data-completar]', (boton) => this.establecerHecho(boton, true));
    this.alHacerClic('[data-reabrir]', (boton) => this.establecerHecho(boton, false));
    this.alHacerClic('[data-eliminar]', (boton) => this.eliminar(boton));
  }

  mostrarFormulario() {
    const form = this.$('[data-formulario="recordatorio"]');
    if (form) form.reset();
  }

  async crear(form) {
    const datos = new FormData(form);
    const cuerpo = {
      titulo: String(datos.get('titulo') || '').trim(),
      detalle: String(datos.get('detalle') || '').trim(),
      prioridad: String(datos.get('prioridad') || 'MEDIA'),
      fecha_limite: String(datos.get('fecha_limite') || ''),
      cuenta_id: Number(datos.get('cuenta_id') || 0) || 0,
    };
    try {
      const creado = await api.crear('/api/recordatorios', cuerpo);
      this.pendientes.push(creado);
      this.pintar();
      form.reset();
      exito('Recordatorio guardado');
    } catch (error) {
      problema(error.message);
    }
  }

  async establecerHecho(boton, hecho) {
    const id = Number(boton.dataset.completar || boton.dataset.reabrir);
    try {
      await api.parchear('/api/recordatorios', { id, hecho });
    } catch (error) {
      problema(error.message);
      return;
    }
    const de = hecho ? this.pendientes : this.completados;
    const hacia = hecho ? this.completados : this.pendientes;
    const i = de.findIndex((r) => r.id === id);
    if (i !== -1) hacia.push(de.splice(i, 1)[0]);
    this.pintar();
    exito(hecho ? 'Recordatorio completado' : 'Recordatorio reabierto');
  }

  async eliminar(boton) {
    const id = Number(boton.dataset.eliminar);
    const item = this.pendientes.find((r) => r.id === id) ||
      this.completados.find((r) => r.id === id);
    if (!item) return;
    if (!window.confirm(`¿Eliminar el recordatorio "${item.titulo}"?`)) return;
    try {
      await api.eliminar(`/api/recordatorios?id=${id}`);
    } catch (error) {
      problema(error.message);
      return;
    }
    this.pendientes = this.pendientes.filter((r) => r.id !== id);
    this.completados = this.completados.filter((r) => r.id !== id);
    this.pintar();
    exito('Recordatorio eliminado');
  }

  tarjetaRecordatorio(r, completado) {
    const cuenta = this.estado.cuentas.find((c) => Number(c.id) === Number(r.cuenta_id));
    const vencido = !r.hecho && r.fecha_limite && r.fecha_limite < hoy();
    const etiqueta = {
      ALTA: ['alta', 'Alta'],
      MEDIA: ['media', 'Media'],
      BAJA: ['baja', 'Baja'],
    }[r.prioridad] || ['media', r.prioridad];
    return `
      <div class="elemento ${completado ? 'cumplido' : ''}">
        <div>
          <strong>${escapar(r.titulo)}</strong>
          ${r.detalle ? `<small>${escapar(r.detalle)}</small>` : ''}
          <small class="tenue">
            <span class="etiqueta ${etiqueta[0]}">${escapar(etiqueta[1])}</span>
            ${r.fecha_limite ? `Límite: <strong class="${vencido ? 'negativo' : ''}">${fecha(r.fecha_limite)}</strong>${vencido ? ' · vencido' : ''}` : 'Sin fecha límite'}
            ${cuenta ? ` · Cuenta ${escapar(cuenta.numero)}` : ''}
            ${r.usuario ? ` · creó ${escapar(r.usuario)}` : ''}
            ${r.hecho && r.fecha_realizado ? `<br>Completado ${fechaHora(r.fecha_realizado)}${r.realizado_por ? ` por ${escapar(r.realizado_por)}` : ''}` : ''}
          </small>
        </div>
        <div class="acciones">
          ${completado && this.estado.puede('movimientos') ? `
            <button type="button" class="secundario" data-reabrir="${r.id}">Reabrir</button>` : ''}
          ${!completado && this.estado.puede('movimientos') ? `
            <button type="button" data-completar="${r.id}">Completar</button>` : ''}
          ${this.estado.puede('movimientos') ? `
            <button type="button" class="peligro" data-eliminar="${r.id}">Eliminar</button>` : ''}
        </div>
      </div>`;
  }

  pintar() {
    const zonaPendientes = this.$('[data-zona="pendientes"]');
    const zonaCompletados = this.$('[data-zona="completados"]');
    const vencidos = this.pendientes.filter((r) => r.fecha_limite && r.fecha_limite < hoy()).length;
    const tituloPendientes = vencidos ? `Pendientes · ${vencidos} vencido(s)` : 'Pendientes';
    this.pintarTitulo(zonaPendientes, tituloPendientes, this.pendientes,
      'No hay pendientes. Anota el próximo compromiso arriba.');
    zonaPendientes.innerHTML += this.pendientes.map((r) => this.tarjetaRecordatorio(r, false)).join('');
    zonaCompletados.innerHTML = this.completados.length
      ? this.completados.map((r) => this.tarjetaRecordatorio(r, true)).join('')
      : '<div class="vacio">Todavía no se ha completado ningún recordatorio.</div>';
  }

  pintarTitulo(zona, titulo, items, vacio) {
    zona.innerHTML = `
      <div class="encabezado-lista">
        <strong>${titulo}</strong>
        <span class="tenue">${items.length} registro(s)</span>
      </div>
      ${items.length ? '' : `<div class="vacio">${vacio}</div>`}`;
  }

  async actualizar() {
    try {
      const [pendientes, completados] = await Promise.all([
        api.obtener('/api/recordatorios?hecho=false'),
        api.obtener('/api/recordatorios?hecho=true'),
      ]);
      this.pendientes = pendientes;
      this.completados = completados;
      this.pintar();
    } catch (error) {
      problema(error.message);
    }
  }
}