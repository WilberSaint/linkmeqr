// Human-readable labels for the raw action codes the backend writes to
// audit_logs (see every audit.Log(...) call across backend/internal/handlers)
// — used anywhere an admin reads the audit trail (the full log and the
// dashboard's "Actividad reciente" widget), so an action like
// "create_print_card" reads as "Tarjeta impresa creada" instead of the raw
// snake_case code a customer never typed.
export const AUDIT_ACTION_LABELS: Record<string, string> = {
  create_template: 'Plantilla creada',
  update_template: 'Plantilla actualizada',
  delete_template: 'Plantilla eliminada',
  activate_template: 'Plantilla activada',
  deactivate_template: 'Plantilla desactivada',
  create_client_profile: 'Perfil de cliente creado',
  create_print_card: 'Tarjeta impresa creada',
  update_print_card: 'Tarjeta impresa actualizada',
  delete_print_card: 'Tarjeta impresa eliminada',
  update_print_card_status: 'Estado de venta actualizado',
  update_loyalty_program: 'Programa de lealtad actualizado',
  loyalty_manual_stamp: 'Sello manual agregado',
  loyalty_redeem: 'Premio de lealtad canjeado',
  activate_license: 'Licencia activada',
  generate_activation_code: 'Código de activación generado',
  generate_activation_code_batch: 'Lote de códigos generado',
  admin_activate_client_license: 'Licencia de cliente activada',
  revoke_activation_code: 'Código de activación revocado',
  create_client: 'Cliente creado',
  update_client: 'Cliente actualizado',
  activate_client: 'Cliente activado',
  deactivate_client: 'Cliente desactivado',
  impersonate_start: 'Suplantación iniciada',
}

export function auditActionLabel(action: string): string {
  return AUDIT_ACTION_LABELS[action] ?? action
}

export const AUDIT_ENTITY_LABELS: Record<string, string> = {
  template: 'plantilla',
  profile: 'perfil',
  print_card: 'tarjeta impresa',
  loyalty_program: 'programa de lealtad',
  loyalty_customer: 'cliente de lealtad',
  license: 'licencia',
  activation_code: 'código de activación',
  activation_code_batch: 'lote de códigos',
  user: 'usuario',
}

export function auditEntityLabel(entityType: string): string {
  return AUDIT_ENTITY_LABELS[entityType] ?? entityType.replace(/_/g, ' ')
}
