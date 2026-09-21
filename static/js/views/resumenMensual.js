import { Vista } from '../core/vista.js';
import { api, consulta, descargar } from '../core/api.js';
import { escapar, opcionesMes } from '../core/dom.js';
import { dinero, periodo } from '../core/formato.js';
import { problema } from '../core/notificaciones.js';

// Resumen mensual: una fila por cuenta con el saldo con que arrancó el mes,
// lo que entró, lo que salió y el cierre. Es el documento de la comisión.
export class ResumenMensual extends Vista {
  static titulo = 'Resumen mensual';
  static glifo = '∑';
  static descripcion = 'Fotografía del mes por cuenta, lista para imprimir.';

  plantilla() {
    const ahora = new Date();
    return `
      ${this.encabezado('Resumen mensual por cuenta', 'Saldo inicial, movimientos del mes y saldo final de todas las cuentas.', `
        <button type="button" class="secundario" data-accion="imprimir">Imprimir</button>
        <button type="button" class="secundario" data-accion="csv">Exportar CSV</button>` )}

      <div class="filtros">
        <label>Mes<select data-campo="mes">${opcionesMes(ahora.getMonth() + 1)}</select></label>
        <label>Año<input type="number" data-campo="anio" min="2000" max="2100" value="${ahora.getFullYear()}"></label>
        <button type="button" data-accion="consultar">Consultar</button>
      </div>

      <div class="tarjeta">
        <div class="documento" data-zona="documento"></div>
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
    return {
      mes: this.$('[data-campo="mes"]').value,
      anio: this.$('[data-campo="anio"]').value || new Date().getFullYear(),
    };
  }

  exportar() {
    descargar(`/api/reportes/resumen-mensual.csv?${consulta(this.filtros())}`)
      .catch((error) => problema(error.message));
  }

  async actualizar() {
    try {
      const datos = await api.obtener(`/api/reportes/resumen-mensual?${consulta(this.filtros())}`);
      this.pintar(datos);
    } catch (error) {
      problema(error.message);
    }
  }

  pintar(datos) {
    this.$('[data-zona="documento"]').innerHTML = `
      <h1>${escapar(this.estado.cooperativa?.nombre || 'Cooperativa')}</h1>
      <p>Resumen mensual de cuentas bancarias · ${escapar(periodo(datos.mes, datos.anio))}</p>`;

    const t = datos.totales;
    this.$('[data-zona="tabla"]').innerHTML = datos.filas.length ? `
      <table>
        <thead><tr>
          <th>Cuenta</th><th>Banco</th><th>Cooperativa</th><th>NIT</th>
          <th class="cifra">Saldo inicial</th><th class="cifra">Depósitos</th>
          <th class="cifra">Cheques</th><th class="cifra">Saldo final</th>
        </tr></thead>
        <tbody>${datos.filas.map((f) => `
          <tr>
            <td><strong>${escapar(f.numero)}</strong><br><small class="tenue">${escapar(f.nombre)}</small></td>
            <td>${escapar(f.banco)}</td>
            <td>${escapar(f.cooperativa)}</td>
            <td>${escapar(f.nit)}</td>
            <td class="cifra">${dinero(f.saldo_inicial_periodo)}</td>
            <td class="cifra deposito">${dinero(f.depositos)}</td>
            <td class="cifra cheque">${dinero(f.cheques)}</td>
            <td class="cifra ${f.saldo_final < 0 ? 'negativo' : ''}">${dinero(f.saldo_final)}</td>
          </tr>`).join('')}</tbody>
        <tfoot><tr>
          <td colspan="4">Totales</td>
          <td class="cifra"></td>
          <td class="cifra">${dinero(t.depositos)}</td>
          <td class="cifra">${dinero(t.cheques)}</td>
          <td class="cifra ${t.saldo_final < 0 ? 'negativo' : ''}">${dinero(t.saldo_final)}</td>
        </tr></tfoot>
      </table>` : '<div class="vacio">Registra cuentas para armar el resumen del mes.</div>';
  }
}