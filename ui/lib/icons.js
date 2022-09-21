export const icons = [
  'Mary Baker',
  'Amelia Earhart',
  'Bessie Coleman',
  'Carrie Chapman',
  'Daisy Gatson',
  'Eunice Kennedy',
  'Felisa Rincon',
  'Gertrude Stein',
  'Hetty Green',
  'Ida B',
  'Jane Johnston',
  'Pearl Kendrick',
  'Lyda Conley',
  'Maya Angelou',
  'Nellie Bly',
  'Georgia O',
  'Pauli Murray',
  'Queen Lili',
  'Rebecca Crumpler',
  'Susan B',
  'Sojourner Truth',
  'Alice Paul',
  'Virginia Apgar',
  'Wilma Rudolph',
  'Dorothea Dix',
  'Rosalyn Yalow',
  'Zora Neale',
]

export function getAvatarName(authName) {
  // check special character
  const spCharsRegExp = /^[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+/
  if (spCharsRegExp.test(authName)) {
    return icons[0]
  }

  // check number
  if (/^\d/.test(authName)) {
    return icons[authName.charCodeAt(0)]
  }

  // check letter
  const order = authName.toLowerCase().charCodeAt(0) - 96

  if (order > 0 && order <= 26) {
    return icons[order]
  }
  return icons[0]
}
