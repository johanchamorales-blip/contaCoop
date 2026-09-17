import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar } from '../core/dom.js';
import { dinero, MESES } from '../core/formato.js';
import { problema } from '../core/notificaciones.js';

// Reporte anual: la evolución mes a mes de una cuenta durante el año, con su
// saldo corrido. Acompaña los informes de la asamblea.
export class ReporteAnual extends Vista {
  static titulo = 'Reporte anual';
  static glifo = '▒';
  static descripcion = 'Comportamiento del año de una cuenta, mes por mes.';
  static necesitaCuenta = true;

  plantilla() {
    const ahora = new Date();
    return `
      ${this.encabezado('Reporte anual', 'Depósitos, cheques y saldo de cada mes del año.', `
        <button type="button" class="secundario" data-accion="imprimir">Imprimir</button>
        <button type="button" class="secundario" data-accion="csv">Exportar CSV</button>` )}

      <div class="filtros">
        <label>Año<input type="number" data-campo="anio" min="2000" max="2100" value="${ahora.getFullYear()}"></label>
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

  filtros() {
    return { anio: this.$('[data-campo="anio"]').value || new Date().getFullYear() };
  }

  exportar() {
    const cuenta = this.estado.cuenta;
    if (!cuenta) {
      problema('Selecciona una cuenta para exportar el reporte anual.');
      return;
    }
    window.location.href = `/api/reportes/anual.csv?${consulta({ cuenta_id: cuenta.id, ...this.filtros() })}`;
  }

  async actualizar() {
    const cuenta = this.estado.cuenta;
    const tabla = this.$('[data-zona="tabla"]');
    if (!cuenta) {
      tabla.innerHTML = '<div class="vacio">Elige una cuenta en la barra de contexto para ver su reporte anual.</div>';
      this.$('[data-zona="documento"]').innerHTML = '';
      this.$('[data-zona="metricas"]').innerHTML = '';
      return;
    }
    try {
      const datos = await api.obtener(`/api/reportes/anual?${consulta({ cuenta_id: cuenta.id, ...this.filtros() })}`);
      this.pintar(datos, cuenta);
    } catch (error) {
      problema(error.message);
    }
  }

  pintar(datos, cuenta) {
    this.$('[data-zona="documento"]').innerHTML = `
      <h1>${escapar(cuenta.cooperativa_id ? this.estado.cooperativa?.nombre || 'Cooperativa' : 'Cooperativa')}</h1>
      <p>Reporte anual · ${escapar(datos.numero)} ${escapar(datos.nombre)} · ${escapar(datos.banco)}</p>
      <p>Año ${datos.anio} · Saldo inicial ${dinero(datos.saldo_inicial_anio)}</p>`;

    this.$('[data-zona="metricas"]').innerHTML = `
      <div class="metrica"><span>Saldo inicial del año</span><strong class="cifra">${dinero(datos.saldo_inicial_anio)}</strong></div>
      <div class="metrica"><span>Depósitos del año</span><strong class="cifra deposito">${dinero(datos.totales_depositos)}</strong></div>
      <div class="metrica"><span>Cheques del año</span><strong class="cifra cheque">${dinero(datos.totales_cheques)}</strong></div>
      <div class="metrica"><span>Saldo a diciembre</span><strong class="cifra ${datos.saldo_final < 0 ? 'negativo' : ''}">${dinero(datos.saldo_final)}</strong></div>`;

    this.$('[data-zona="tabla"]').innerHTML = `
      <table>
        <thead><tr>
          <th>Mes</th><th class="cifra">Depósitos</th>
          <th class="cifra">Cheques</th><th class="cifra">Saldo final</th>
        </tr></thead>
        <tbody>${datos.filas.map((f) => `
          <tr>
            <td>${MESES[f.mes - 1]}</td>
            <td class="cifra deposito">${dinero(f.depositos)}</td>
            <td class="cifra cheque">${dinero(f.cheques)}</td>
            <td class="cifra ${f.saldo_final < 0 ? 'negativo' : ''}">${dinero(f.saldo_final)}</td>
          </tr>`).join('')}</tbody>
        <tfoot><tr>
          <td>Totales</td>
          <td class="cifra">${dinero(datos.totales_depositos)}</td>
          <td class="cifra">${dinero(datos.totales_cheques)}</td>
          <td class="cifra">${dinero(datos.saldo_final)}</td>
        </tr></tfoot>
      </table>`;
  }
}