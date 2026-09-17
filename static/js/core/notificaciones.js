// Mensajes breves en la esquina inferior. Duran lo suficiente para leerse
// y no bloquean la pantalla como hacía alert().
const contenedor = document.createElement('div');
contenedor.className = 'notificaciones';
contenedor.setAttribute('role', 'status');
contenedor.setAttribute('aria-live', 'polite');
document.body.appendChild(contenedor);

// Tope de notificaciones visibles a la vez: si el usuario se mueve rápido
// entre módulos, las más viejas se retiran en vez de seguir apilándose.
const MAX_NOTIFICACIONES = 4;

export function notificar(mensaje, tono = 'neutro', duracion = 4200) {
  while (contenedor.children.length >= MAX_NOTIFICACIONES) {
    contenedor.firstElementChild?.remove();
  }

  const nodo = document.createElement('div');
  nodo.className = `notificacion ${tono}`;

  const texto = document.createElement('span');
  texto.textContent = mensaje;
  nodo.appendChild(texto);

  const cerrar = document.createElement('button');
  cerrar.type = 'button';
  cerrar.className = 'notificacion-cerrar';
  cerrar.setAttribute('aria-label', 'Cerrar aviso');
  cerrar.textContent = '×';

  let temporizador = setTimeout(quitar, duracion);
  function quitar() {
    clearTimeout(temporizador);
    nodo.remove();
  }
  cerrar.addEventListener('click', quitar);
  nodo.appendChild(cerrar);

  contenedor.appendChild(nodo);
}

export const exito = (mensaje) => notificar(mensaje, 'logro');
export const problema = (mensaje) => notificar(mensaje, 'error', 6000);
export const pendiente = (mensaje) => notificar(mensaje, 'pendiente', 7000);
