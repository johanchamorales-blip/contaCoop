// Cliente HTTP del sistema. Centraliza el manejo de errores para que las
// vistas solo se ocupen de qué pedir, no de cómo.
const oyentesSinSesion = new Set();

export function alPerderSesion(fn) {
  oyentesSinSesion.add(fn);
}

export class ErrorAPI extends Error {
  constructor(mensaje, estado) {
    super(mensaje);
    this.estado = estado;
  }
}

async function peticion(url, opciones = {}) {
 const urlCompleta = url;
  const respuesta = await fetch(urlCompleta, {
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    ...opciones,
  });

  const tipo = respuesta.headers.get('content-type') || '';
  const cuerpo = tipo.includes('application/json') ? await respuesta.json() : await respuesta.text();

  if (respuesta.status === 401 && !url.includes('/api/auth/login') && !url.includes('/api/auth/sesion')) {
      oyentesSinSesion.forEach((fn) => fn());
      throw new ErrorAPI('Tu sesión expiró. Vuelve a iniciar sesión.', 401);
    }
  if (!respuesta.ok) {
    const mensaje = (cuerpo && cuerpo.error) || 'No se pudo completar la operación';
    throw new ErrorAPI(mensaje, respuesta.status);
  }
  return cuerpo;
}

export const api = {
  obtener: (url) => peticion(url),
  crear: (url, datos) => peticion(url, { method: 'POST', body: JSON.stringify(datos) }),
  actualizar: (url, datos) => peticion(url, { method: 'PUT', body: JSON.stringify(datos) }),
  parchear: (url, datos) => peticion(url, { method: 'PATCH', body: JSON.stringify(datos) }),
  eliminar: (url) => peticion(url, { method: 'DELETE' }),
};

// Descarga un archivo (CSV o Excel) con la sesión actual, sin abandonar la
// vista. Usa credenciales explícitas para que la cookie de sesión siempre
// viaje, igual que el resto de peticiones de api.js.
export async function descargar(url) {
  const respuesta = await fetch(url, { credentials: 'include' });
  if (respuesta.status === 401) {
    oyentesSinSesion.forEach((fn) => fn());
    throw new ErrorAPI('Tu sesión expiró. Vuelve a iniciar sesión.', 401);
  }
  if (!respuesta.ok) {
    throw new ErrorAPI('No se pudo descargar el archivo', respuesta.status);
  }
  const blob = await respuesta.blob();
  const disposicion = respuesta.headers.get('content-disposition') || '';
  const coincidencia = disposicion.match(/filename="?([^";]+)"?/);
  const nombre = coincidencia ? coincidencia[1] : 'descarga';
  const objeto = window.URL.createObjectURL(blob);
  const enlace = document.createElement('a');
  enlace.href = objeto;
  enlace.download = nombre;
  document.body.appendChild(enlace);
  enlace.click();
  enlace.remove();
  window.URL.revokeObjectURL(objeto);
  return nombre;
}

// Construye una query string omitiendo valores vacíos.
export function consulta(parametros) {
  const params = new URLSearchParams();
  Object.entries(parametros).forEach(([clave, valor]) => {
    if (valor !== undefined && valor !== null && valor !== '') params.set(clave, valor);
  });
  return params.toString();
}
