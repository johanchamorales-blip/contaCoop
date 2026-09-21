import { Vista } from '../core/vista.js';
import { api, consulta } from '../core/api.js';
import { escapar, datosFormulario, opcionesMes } from '../core/dom.js';
import { dinero, fecha, periodo } from '../core/formato.js';
import { exito, problema, pendiente } from '../core/notificaciones.js';

// URL base de tu backend Go en Railway
const API_BASE_URL = "";

// Conciliación bancaria: se calcula primero y solo se guarda si el usuario
// acepta el resultado. El lado del banco y el de los libros se muestran por
// separado para que la diferencia sea evidente.
export class Conciliaciones extends Vista {
  static titulo = 'Conciliación bancaria';
  static glifo = '⇌';
  static necesitaCuenta = true;

  constructor(contexto) {
    super(contexto);
    this.calculo = null;
  }

  plantilla() {
    const ahora = new Date();
    const mesAnterior = ahora.getMonth() === 0 ? 12 : ahora.getMonth();
    const anio = ahora.getMonth() === 0 ? ahora.getFullYear() - 1 : ahora.getFullYear();
    return `
      ${this.encabezado('Conciliación bancaria', 'Los cheques emitidos sin cobrar se detectan solos como cheques en circulación.', `
        <button type="button" class="secundario" data-accion="imprimir">Imprimir</button>
        <button type="button" class="secundario" data-accion="descargar">Descargar Excel</button>
        <button type="button" class="secundario" data-accion="csv">Descargar CSV</button>`)}

      <div class="tarjeta no-imprimir">
        <h2>Datos del estado de cuenta</h2>
        <form class="formulario" data-formulario="conciliacion" style="margin-top:14px">
          <div class="par">
            <label>Mes<select name="mes">${opcionesMes(mesAnterior)}</select></label>
            <label>Año<input name="anio" type="number" min="2000" max="2100" value="${anio}" required></label>
          </div>
          <div class="par">
            <label>Saldo según estado de cuenta
              <span class="par"><input name="saldo_estado_cuenta" type="number" step="0.01" required placeholder="0.00"><input name="fecha_saldo_estado_cuenta" type="date"></span>
            </label>
            <label>Depósitos en tránsito
              <span class="par"><input name="depositos_en_transito" type="number" step="0.01" min="0" value="0"><input name="fecha_depositos_en_transito" type="date"></span>
            </label>
          </div>
          <div class="par">
            <label>Notas de débito
              <span class="par"><input name="notas_debito" type="number" step="0.01" min="0" value="0"><input name="fecha_notas_debito" type="date"></span>
            </label>
            <label>Notas de crédito
              <span class="par"><input name="notas_credito" type="number" step="0.01" min="0" value="0"><input name="fecha_notas_credito" type="date"></span>
            </label>
          </div>
          <label class="ancho-total">Otros ajustes
              <span class="ajuste-campos">
                <input name="ajuste_nombre" type="text" placeholder="Concepto del ajuste (ej. comisión no registrada, error en cheque #102)">
                <select name="ajuste_tipo">
                  <option value="">Sin ajuste</option>
                  <option value="LIBROS_DEBE">Suma a libros (Debe)</option>
                  <option value="LIBROS_HABER">Resta de libros (Haber)</option>
                  <option value="BANCO_DEBE">Suma al banco (Debe)</option>
                  <option value="BANCO_HABER">Resta del banco (Haber)</option>
                </select>
                <input name="ajuste_monto" type="number" step="0.01" min="0" placeholder="Monto">
                <input name="ajuste_fecha" type="date">
              </span>
            </label>
          <label class="ancho-total">Fecha y lugar<input name="fecha_lugar" placeholder="Guatemala, 31/08/2026"></label>
        <label class="ancho-total">Observaciones<textarea name="observaciones" rows="2" placeholder="Notas para la comisión de vigilancia"></textarea></label>
          <div class="acciones">
            <button type="submit">Calcular conciliación</button>
            <button type="button" data-accion="guardar" disabled>Guardar conciliación</button>
          </div>
        </form>
      </div>

      <div class="tarjeta" data-zona="resultado" hidden></div>

      <div class="tarjeta no-imprimir">
        <h2>Conciliaciones guardadas</h2>
        <div class="lista" data-zona="historial" style="margin-top:14px"></div>
      </div>`;
  }

  conectar() {
    this.alEnviar('[data-formulario="conciliacion"]', (form) => this.calcular(form));
    this.alHacerClic('[data-accion="guardar"]', () => this.guardar());
    this.alHacerClic('[data-accion="imprimir"]', () => this.imprimir());
    this.alHacerClic('[data-accion="descargar"]', () => this.descargar());
    this.alHacerClic('[data-accion="csv"]', () => this.descargarCSV());
    this.alHacerClic('[data-ver-conciliacion]', (boton) => this.verGuardada(Number(boton.dataset.verConciliacion)));
  }

  datos() {
    const form = this.$('[data-formulario="conciliacion"]');
    const datos = datosFormulario(form);
    return {
      mes: Number(datos.mes),
      anio: Number(datos.anio),
      saldo_estado_cuenta: Number(datos.saldo_estado_cuenta || 0),
      depositos_en_transito: Number(datos.depositos_en_transito || 0),
      notas_debito: Number(datos.notas_debito || 0),
      notas_credito: Number(datos.notas_credito || 0),
      ajuste_nombre: datos.ajuste_nombre || '',
      ajuste_tipo: datos.ajuste_tipo || '',
      ajuste_monto: Number(datos.ajuste_monto || 0),
      fecha_saldo_estado_cuenta: datos.fecha_saldo_estado_cuenta || '',
      fecha_depositos_en_transito: datos.fecha_depositos_en_transito || '',
      fecha_notas_debito: datos.fecha_notas_debito || '',
      fecha_notas_credito: datos.fecha_notas_credito || '',
      fecha_ajustes: datos.fecha_ajustes || '',
      fecha_lugar: datos.fecha_lugar || '',
      observaciones: datos.observaciones || '',
    };
  }

  async calcular() {
    const cuenta = this.estado.cuenta;
    if (!cuenta) {
      problema('Selecciona una cuenta antes de conciliar.');
      return;
    }
    try {
      const resultado = await api.obtener(`/api/conciliacion?${consulta({ cuenta_id: cuenta.id, ...this.datos() })}`);
      this.calculo = resultado;
      this.pintarResultado(resultado, false);
      this.$('[data-accion="guardar"]').disabled = false;
    } catch (error) {
      problema(error.message);
    }
  }

  async guardar() {
    const cuenta = this.estado.cuenta;
    if (!this.calculo) return;
    try {
      const guardada = await api.crear('/api/conciliaciones', { cuenta_id: cuenta.id, ...this.datos() });
      this.calculo = guardada;
      this.pintarResultado(guardada, false);
      this.$('[data-accion="guardar"]').disabled = true;
      exito('Conciliación guardada');
      if (!guardada.cuadrada) pendiente('Quedó guardada con diferencia; revísala con el banco.');
      await this.cargarHistorial();
    } catch (error) {
      problema(error.message);
    }
  }

  pintarResultado(resultado, guardada = false) {
    const r = resultado.reporte;
    const zona = this.$('[data-zona="resultado"]');
    const fechaItem = (v) => (v ? `<small class="tenue">${fecha(v)}</small>` : '');
    const ajusteLibros = Number(r.ajustes) || 0;
    const ajusteBanco = Number(r.ajuste_banco) || 0;
    const ajusteNombre = r.ajuste_nombre ? ` · ${escapar(r.ajuste_nombre)}` : '';
    const lineaAjuste = (valor) => {
      const positivo = valor >= 0;
      return `<div class="metrica"><span>(±) Ajuste${ajusteNombre}</span><strong class="cifra ${positivo ? 'deposito' : 'cheque'}">${positivo ? '+' : '−'}${dinero(Math.abs(valor))}</strong>${fechaItem(r.fecha_ajustes)}</div>`;
    };
    zona.hidden = false;
    zona.innerHTML = `
      <div class="encabezado-vista secundario-contenido">
        <div>
          <h2>${guardada ? `Conciliación guardada #${r.id}` : 'Resultado de la conciliación'} · ${escapar(periodo(r.mes, r.anio))}</h2>
          ${r.usuario ? `<p>Elaboró: ${escapar(r.usuario)}</p>` : ''}
        </div>
        <div class="acciones no-imprimir">
          ${guardada ? `<button type="button" class="secundario" data-accion="descargar-detalle">Descargar Excel</button>` : ''}
          <button type="button" data-accion="imprimir-resultado">Imprimir esta conciliación</button>
        </div>
      </div>
      <div class="aviso ${resultado.cuadrada ? 'logro' : 'pendiente'}">
        <span aria-hidden="true">${resultado.cuadrada ? '✓' : '▲'}</span>
        <span>${resultado.cuadrada
          ? 'Los libros y el banco cuadran al centavo.'
          : `Diferencia de ${dinero(r.diferencia_con_libros)} entre libros y banco.`}</span>
      </div>
      ${(resultado.alertas || []).filter((a) => a.codigo !== 'DIFERENCIA').map((a) => `
        <div class="aviso pendiente"><span aria-hidden="true">▲</span><span>${escapar(a.mensaje)}</span></div>`).join('')}

      <div class="columnas" style="margin-top:6px">
        <div>
          <h3>Según el banco</h3>
          <div class="metricas" style="margin-top:8px">
            <div class="metrica"><span>Estado de cuenta</span><strong class="cifra">${dinero(r.saldo_estado_cuenta)}</strong>${fechaItem(r.fecha_saldo_estado_cuenta)}</div>
            <div class="metrica"><span>(+) Depósitos en tránsito</span><strong class="cifra deposito">${dinero(r.depositos_en_transito)}</strong>${fechaItem(r.fecha_depositos_en_transito)}</div>
            <div class="metrica"><span>(−) Cheques en circulación</span><strong class="cifra cheque">${dinero(r.cheques_circulacion)}</strong></div>
            ${ajusteBanco ? lineaAjuste(ajusteBanco) : ''}
            <div class="metrica"><span>Saldo bancario ajustado</span><strong class="cifra">${dinero(r.saldo_conciliado)}</strong></div>
          </div>
        </div>
        <div>
          <h3>Según los libros</h3>
          <div class="metricas" style="margin-top:8px">
            <div class="metrica"><span>Saldo en libros</span><strong class="cifra">${dinero(r.saldo_libros)}</strong></div>
            <div class="metrica"><span>(+) Notas de crédito</span><strong class="cifra deposito">${dinero(r.notas_credito)}</strong>${fechaItem(r.fecha_notas_credito)}</div>
            <div class="metrica"><span>(−) Notas de débito</span><strong class="cifra cheque">${dinero(r.notas_debito)}</strong>${fechaItem(r.fecha_notas_debito)}</div>
            ${lineaAjuste(ajusteLibros)}
            <div class="metrica"><span>Saldo en libros ajustado</span><strong class="cifra">${dinero(r.saldo_libros_ajustado)}</strong></div>
          </div>
        </div>
      </div>

      <h3 style="margin-top:16px">Cheques en circulación al cierre</h3>
      <div class="tabla-marco" style="margin-top:8px">
        ${(resultado.cheques_en_circulacion || []).length ? `
          <table>
            <thead><tr><th>Fecha</th><th>Cheque</th><th>Beneficiario</th><th>Concepto</th><th class="cifra">Monto</th></tr></thead>
            <tbody>${resultado.cheques_en_circulacion.map((c) => `
              <tr>
                <td>${fecha(c.fecha_operacion)}</td>
                <td>${escapar(c.numero_documento)}</td>
                <td>${escapar(c.beneficiario || '')}</td>
                <td>${escapar(c.concepto)}</td>
                <td class="cifra cheque">${dinero(c.monto)}</td>
              </tr>`).join('')}</tbody>
          </table>` : '<div class="vacio">Ningún cheque quedó pendiente de cobro en el periodo.</div>'}
      </div>
      <div class="firmas">
        <div>Elaboró</div><div>Tesorero</div><div>Vo. Bo.</div><div>Presidente Comisión de Vigilancia</div>
      </div>`;

    const botonImprimir = zona.querySelector('[data-accion="imprimir-resultado"]');
    if (botonImprimir) botonImprimir.addEventListener('click', () => this.imprimir());
    const botonDescargar = zona.querySelector('[data-accion="descargar-detalle"]');
    if (botonDescargar) botonDescargar.addEventListener('click', () => this.descargar());
  }

  async verGuardada(id) {
    if (!id) return;
    try {
      const resultado = await api.obtener(`/api/conciliaciones/${id}`);
      this.calculo = resultado;
      this.$('[data-accion="guardar"]').disabled = true;
      this.pintarResultado(resultado, true);
      this.$('[data-zona="resultado"]').scrollIntoView({ behavior: 'smooth', block: 'start' });
    } catch (error) {
      problema(error.message);
    }
  }

  imprimir() {
    this.descargar();
  }

  descargar() {
    if (!this.calculo?.reporte?.id) {
      problema('Guarda la conciliación antes de descargarla.');
      return;
    }
    window.location.href = `${API_BASE_URL}/api/conciliaciones.xlsx?id=${encodeURIComponent(this.calculo.reporte.id)}`;
  }

  async descargarCSV() {
    if (!this.calculo?.reporte?.id) {
      problema('Guarda la conciliación antes de descargarla.');
      return;
    }
    
    // Mostrar URL tentativo para depuración
    const url = `${API_BASE_URL}/api/conciliaciones.csv?id=${encodeURIComponent(this.calculo.reporte.id)}`;
    console.log(" Intentando descargar desde:", url);
    
    try {
      const response = await fetch(url);
      console.log(" Respuesta del servidor:", response.status, response.statusText);
      
      if (!response.ok) {
        // Si es 404, dar instrucción específica
        if (response.status === 404) {
          problema(` Error 404: El servidor no encontró la ruta.\n\nProbable causa:\n- El backend en Render está dormido\n- Intenta entrar a https://contacoop.onrender.com primero para despertarlo\n- Luego vuelve a hacer clic en CSV`);
          return;
        }
        throw new Error(`HTTP ${response.status}`);
      }
      
      const blob = await response.blob();
      const objUrl = window.URL.createObjectURL(blob);
      
      // Disparar descarga en segundo plano
      const a = document.createElement('a');
      a.href = objUrl;
      a.download = `conciliacion_${this.calculo.reporte.id}.csv`;
      document.body.appendChild(a);
      a.click();
      
      // Limpieza inmediata
      a.remove();
      window.URL.revokeObjectURL(objUrl);
      
      console.log("✅ Descarga iniciada exitosamente");
    } catch (error) {
      console.error("❌ Error en descarga CSV:", error);
      problema(`Error de conexión: ${error.message}\n\nVerifica la consola (F12) para más detalles.`);
    }
  }

  async cargarHistorial() {
    const cuenta = this.estado.cuenta;
    const zona = this.$('[data-zona="historial"]');
    if (!cuenta) {
      zona.innerHTML = '<div class="vacio">Selecciona una cuenta para ver su historial.</div>';
      return;
    }
    const items = await api.obtener(`/api/conciliaciones?cuenta_id=${cuenta.id}`);
    zona.innerHTML = items.length ? items.map((item) => `
      <div class="elemento">
        <div>
          <strong>${escapar(periodo(item.mes, item.anio))}</strong>
          <small>Conciliado ${dinero(item.saldo_conciliado)} · libros ${dinero(item.saldo_libros_ajustado || item.saldo_libros)}</small>
          <small>${escapar(item.fecha_lugar || 'Sin fecha y lugar')}${item.usuario ? ` · ${escapar(item.usuario)}` : ''}</small>
        </div>
        <div class="acciones">
          <span class="etiqueta ${Math.abs(item.diferencia_con_libros) < 0.005 ? 'cobrado' : 'emitido'}">
            ${Math.abs(item.diferencia_con_libros) < 0.005 ? 'Cuadrada' : `Dif. ${dinero(item.diferencia_con_libros)}`}
          </span>
          <button type="button" class="secundario" data-ver-conciliacion="${item.id}">Ver</button>
        </div>
      </div>`).join('') : '<div class="vacio">Esta cuenta todavía no tiene conciliaciones guardadas.</div>';
  }

  async actualizar() {
    this.calculo = null;
    this.$('[data-zona="resultado"]').hidden = true;
    this.$('[data-accion="guardar"]').disabled = true;
    await this.cargarHistorial();
  }
}