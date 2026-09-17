import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar, opciones } from '../core/dom.js';
import { dinero, fecha, hoy } from '../core/formato.js';
import { problema } from '../core/notificaciones.js';

// Cheques en circulación: los egresos emitidos que el banco todavía no ha
// cobrado. La federación pide vigilar los que superan 30, 60 y 90 días.
export class ChequesCirculacion extends Vista {
  static titulo = 'Cheques en circulación';
  static glifo = '◌';
  static descripcion = 'Egresos emitidos sin cobrar, ordenados por antigüedad.';

  plantilla() {
    return `
      ${this.encabezado('Cheques en circulación', 'Pendientes de cobro en el banco, ordenados del más antiguo al más reciente.', `
        <button type="button" class="secundario" data-accion="imprimir">Imprimir</button>
        <button type="button" class="secundario" data-accion="csv">Exportar CSV</button>` )}

      <div class="filtros">
        <label>Cuenta
          <select data-campo="cuenta_id">
            ${opciones(this.estado.cuentas, { valor: (c) => c.id, texto: (c) => `${c.numero} · ${c.nombre}`, vacio: 'Todas las cuentas' })}
          </select>
        </label>
        <label>Corte<input type="date" data-campo="corte" value="${hoy()}"></label>
        <button type="button" data-accion="consultar">Consultar</button>
      </div>

      <div class="tarjeta">
        <div class="documento" data-zona="documento"></div>
        <div class="metricas" data-zona="metricas"></div>
        <div class="tabla-marco" style="margin-top:14px" data-zona="tabla"></div>
        <div class="firmas">
          <div>Elaboró</div><div>Tesorero</div><div>Vo. Bo.</div><div>Presidente Comisión de Vigilancia</div>
        </div>
      </div>`;
  }

  conectar() {
    this.alHacerClic('[data-accion="consultar"]', () => this.actualizar());
    this.alHacerClic('[data-accion="imprimir"]', () => window.print());
    this.alHacerClic('[data-accion="csv"]', () => this.exportar());
  }

  parametros() {
    const corte = this.$('[data-campo="corte"]').value;
    const cuenta_id = this.$('[data-campo="cuenta_id"]').value;
    return { corte: corte || hoy(), cuenta_id };
  }

  exportar() {
    window.location.href = `/api/reportes/cheques-en-circulacion.csv?${consulta(this.parametros())}`;
  }

  async actualizar() {
    try {
      const datos = await api.obtener(`/api/reportes/cheques-en-circulacion?${consulta(this.parametros())}`);
      this.pintar(datos);
    } catch (error) {
      problema(error.message);
    }
  }

  pintar(datos) {
    const seleccion = this.$('[data-campo="cuenta_id"]').value;
    const tituloCuenta = this.estado.cuentas.find((c) => String(c.id) === String(seleccion));

    this.$('[data-zona="documento"]').innerHTML = `
      <h1>${escapar(this.estado.cooperativa?.nombre || 'Cooperativa')}</h1>
      <p>Cheques en circulación a la fecha de corte ${fecha(datos.corte)}</p>
      <p>Cuenta: ${tituloCuenta ? escapar(`${tituloCuenta.numero} ${tituloCuenta.nombre}`) : 'Todas las cuentas'}</p>`;

    const viejo = Number(datos.por_edad['90+'].conteo);
    const mediano = Number(datos.por_edad['60-89'].conteo);
    this.$('[data-zona="metricas"]').innerHTML = `
      <div class="metrica"><span>En circulación</span><strong class="cifra">${datos.conteo} cheque(s)</strong><small>${dinero(datos.monto_total)}</small></div>
      <div class="metrica"><span>Más de 90 días</span><strong class="cifra ${viejo ? 'negativo' : ''}">${viejo}</strong><small>${dinero(datos.por_edad['90+'].monto)}</small></div>
      <div class="metrica"><span>De 60 a 89 días</span><strong class="cifra ${mediano ? 'negativo' : ''}">${mediano}</strong><small>${dinero(datos.por_edad['60-89'].monto)}</small></div>
      <div class="metrica"><span>De 30 a 59 días</span><strong class="cifra">${Number(datos.por_edad['30-59'].conteo)}</strong><small>${dinero(datos.por_edad['30-59'].monto)}</small></div>
      <div class="metrica"><span>Menos de 30 días</span><strong class="cifra">${Number(datos.por_edad['0-29'].conteo)}</strong><small>${dinero(datos.por_edad['0-29'].monto)}</small></div>`;

    this.$('[data-zona="tabla"]').innerHTML = datos.cheques.length ? `
      <table>
        <thead><tr>
          <th>Antigüedad</th><th>Fecha</th><th>Cheque</th><th>Beneficiario</th>
          <th>Concepto</th><th>Cuenta</th><th class="cifra">Monto</th>
        </tr></thead>
        <tbody>${datos.cheques.map((c) => `
          <tr>
            <td><span class="etiqueta ${c.rango === '90+' ? 'alta' : c.rango === '60-89' ? 'media' : 'baja'}">${c.dias_en_circulacion} días</span></td>
            <td>${fecha(c.fecha_operacion)}</td>
            <td>${escapar(c.numero_documento)}</td>
            <td>${escapar(c.beneficiario || '')}</td>
            <td>${escapar(c.concepto)}</td>
            <td>${escapar(c.numero_cuenta)}<br><small class="tenue">${escapar(c.banco)}</small></td>
            <td class="cifra cheque">${dinero(c.monto)}</td>
          </tr>`).join('')}</tbody>
        <tfoot><tr>
          <td colspan="6">Total</td>
          <td class="cifra">${dinero(datos.monto_total)}</td>
        </tr></tfoot>
      </table>` : '<div class="vacio">Ningún cheque pendiente de cobro en la fecha de corte.</div>';
  }
}