// Formato de moneda y fechas para Guatemala.
const quetzales = new Intl.NumberFormat('es-GT', {
  style: 'currency',
  currency: 'GTQ',
  minimumFractionDigits: 2,
});

export function dinero(valor) {
  return quetzales.format(Number(valor) || 0);
}

// Monto en letras para el cheque. El formato típico es
// "un mil doscientos treinta y cuatro quetzales con 56/100".
const UNIDADES = [
  '', 'uno', 'dos', 'tres', 'cuatro', 'cinco', 'seis', 'siete', 'ocho', 'nueve',
  'diez', 'once', 'doce', 'trece', 'catorce', 'quince', 'dieciséis', 'diecisiete',
  'dieciocho', 'diecinueve', 'veinte', 'veintiuno', 'veintidós', 'veintitrés',
  'veinticuatro', 'veinticinco', 'veintiséis', 'veintisiete', 'veintiocho', 'veintinueve',
];
const DECENAS = ['', '', '', 'treinta', 'cuarenta', 'cincuenta', 'sesenta', 'setenta', 'ochenta', 'noventa'];
const CENTENAS = ['', 'ciento', 'doscientos', 'trescientos', 'cuatrocientos', 'quinientos', 'seiscientos', 'setecientos', 'ochocientos', 'novecientos'];

function tresCifras(n) {
  const centena = Math.floor(n / 100);
  const resto = n % 100;
  if (centena === 1 && resto === 0) return 'cien';
  let parte = centena ? CENTENAS[centena] : '';
  if (resto >= 30) {
    const decena = Math.floor(resto / 10);
    const unidad = resto % 10;
    const decen = unidad
      ? `${DECENAS[decena]} y ${UNIDADES[unidad]}`
      : DECENAS[decena];
    parte = `${parte ? `${parte} ` : ''}${decen}`;
    return parte;
  }
  if (resto) parte = `${parte ? `${parte} ` : ''}${UNIDADES[resto]}`;
  return parte;
}

// "veintiuno" y "uno" se apocopan cuando preceden a la cantidad: "veintiún mil
// quetzales", "un mil...". Cuando la cifra termina en 1 (21, 121, 1001...)
// también: "ciento veintiún quetzales".
function apocopar(texto) {
  return texto
    .replace('veintiuno mil', 'veintiún mil')
    .replace('uno mil', 'un mil')
    .replace(/veintiuno$/, 'veintiún')
    .replace(/uno$/, 'un');
}

function enteroEnLetras(n) {
  if (n === 0) return 'cero';
  if (n < 1000) return apocopar(tresCifras(n));
  const miles = Math.floor(n / 1000);
  const resto = n % 1000;
  let texto = '';
  if (miles === 1) {
    texto = 'mil';
  } else if (miles < 1000) {
    texto = `${apocopar(tresCifras(miles))} mil`;
  } else {
    const millones = Math.floor(miles / 1000);
    const restoMiles = miles % 1000;
    texto = millones === 1 ? 'un millón' : `${enteroEnLetras(millones)} millones`;
    if (restoMiles) texto += ` ${apocopar(tresCifras(restoMiles))} mil`;
  }
  if (resto) texto += ` ${apocopar(tresCifras(resto))}`;
  return texto;
}

export function montoEnLetras(valor) {
  const cantidad = Math.round(Number(valor) * 100) / 100;
  const entero = Math.floor(Math.abs(cantidad));
  const centavos = Math.round((Math.abs(cantidad) - entero) * 100);
  let texto = entero === 0
    ? 'cero'
    : apocopar(enteroEnLetras(entero));
  const moneda = entero === 1 ? 'quetzal' : 'quetzales';
  if (entero % 1000000 === 0 && entero >= 1000000) texto += ' de';
  return `${texto} ${moneda} con ${String(centavos).padStart(2, '0')}/100`;
}

// Las fechas del servidor vienen como "2026-09-30" (día calendario).
export function fecha(valor) {
  if (!valor) return '';
  const [anio, mes, dia] = String(valor).slice(0, 10).split('-');
  if (!anio || !mes || !dia) return '';
  return `${dia}/${mes}/${anio}`;
}

export function fechaHora(valor) {
  if (!valor) return '';
  const d = new Date(valor);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('es-GT', { dateStyle: 'short', timeStyle: 'short' });
}

export function hoy() {
  const d = new Date();
  const mes = String(d.getMonth() + 1).padStart(2, '0');
  const dia = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${mes}-${dia}`;
}

export const MESES = [
  'Enero', 'Febrero', 'Marzo', 'Abril', 'Mayo', 'Junio',
  'Julio', 'Agosto', 'Septiembre', 'Octubre', 'Noviembre', 'Diciembre',
];

export function periodo(mes, anio) {
  if (!mes || !anio) return 'Histórico completo';
  return `${MESES[mes - 1]} ${anio}`;
}

export function nombreRol(rol) {
  return {
    ADMINISTRADOR: 'Administrador',
    CONTADOR: 'Contador',
    JEFE_DE_PLANTA: 'Jefe de planta',
  }[rol] || rol;
}
