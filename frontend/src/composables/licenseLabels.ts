const LABELS: Record<string, string> = {
  ACTIVE: 'Activa',
  INACTIVE: 'Inactiva',
  EXPIRED: 'Expirada',
}

export function licenseStatusLabel(status: string | undefined | null) {
  if (!status) return LABELS.INACTIVE
  return LABELS[status] ?? status
}

const CODE_LABELS: Record<string, string> = {
  UNUSED: 'Sin usar',
  USED: 'Usado',
  REVOKED: 'Revocado',
}

export function codeStatusLabel(status: string | undefined | null) {
  if (!status) return status
  return CODE_LABELS[status] ?? status
}
