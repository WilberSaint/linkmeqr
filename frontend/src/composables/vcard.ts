import type { Profile, ProfileBlock } from '@/types'

/**
 * Builds a .vcf contact card out of the profile's own blocks.
 *
 * There is no separate "contact details" form to read from — the phone,
 * WhatsApp, email and address a business publishes ARE its blocks, so this
 * mines them back out of the stored URLs (tel:, mailto:, wa.me/…) rather
 * than asking the business to type everything a second time.
 */

type ContactBits = {
  phone?: string
  whatsapp?: string
  email?: string
  website?: string
  address?: string
}

/** Strips everything but digits and a leading +, the shape a dialer wants. */
function cleanPhone(raw: string): string {
  const plus = raw.trim().startsWith('+')
  const digits = raw.replace(/[^\d]/g, '')
  return digits ? (plus ? `+${digits}` : digits) : ''
}

export function contactBitsFrom(blocks: ProfileBlock[]): ContactBits {
  const bits: ContactBits = {}

  for (const b of blocks) {
    const url = (b.url ?? '').trim()
    if (!url) continue

    switch (b.block_type) {
      case 'phone':
        bits.phone ||= cleanPhone(url.replace(/^tel:/i, ''))
        break
      case 'whatsapp': {
        // Stored as a wa.me/<number> or api.whatsapp.com/send?phone=<number>
        // link; either way the number is the only part a contact card wants.
        const m = /(?:wa\.me\/|phone=)(\+?[\d\s()-]+)/i.exec(url)
        if (m) bits.whatsapp ||= cleanPhone(m[1])
        break
      }
      case 'email':
        bits.email ||= url.replace(/^mailto:/i, '').split('?')[0].trim()
        break
      case 'website':
        bits.website ||= url
        break
      case 'location':
        // A maps link is a URL, not an address — the human-readable address
        // is whatever the business titled or described the block with.
        bits.address ||= (b.description || b.title || '').trim()
        break
    }
  }

  return bits
}

/** True when there's enough here for a contact card to be worth offering. */
export function hasContactInfo(blocks: ProfileBlock[]): boolean {
  const b = contactBitsFrom(blocks)
  return Boolean(b.phone || b.whatsapp || b.email)
}

/** Escapes the characters vCard treats as structure. */
function esc(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/;/g, '\\;').replace(/,/g, '\\,').replace(/\n/g, '\\n')
}

export function buildVCard(
  profile: Pick<Profile, 'business_name' | 'description'>,
  blocks: ProfileBlock[],
  publicUrl: string,
): string {
  const bits = contactBitsFrom(blocks)
  const name = profile.business_name || 'Contacto'

  const lines = ['BEGIN:VCARD', 'VERSION:3.0', `FN:${esc(name)}`, `ORG:${esc(name)}`]

  if (bits.phone) lines.push(`TEL;TYPE=WORK,VOICE:${bits.phone}`)
  // Labelled CELL so it lands as a second, distinguishable number rather
  // than silently overwriting the landline on import.
  if (bits.whatsapp && bits.whatsapp !== bits.phone) lines.push(`TEL;TYPE=CELL:${bits.whatsapp}`)
  if (bits.email) lines.push(`EMAIL;TYPE=WORK:${esc(bits.email)}`)
  if (bits.website) lines.push(`URL:${esc(bits.website)}`)
  if (bits.address) lines.push(`ADR;TYPE=WORK:;;${esc(bits.address)};;;;`)
  if (profile.description) lines.push(`NOTE:${esc(profile.description)}`)
  // The profile itself, so the saved contact leads back to the live page
  // even after the printed card is long gone.
  lines.push(`URL;TYPE=LinkMeQR:${esc(publicUrl)}`)

  lines.push('END:VCARD')
  return lines.join('\r\n')
}

/** Triggers the browser's own "save file" path for the generated card. */
export function downloadVCard(filename: string, vcard: string) {
  const blob = new Blob([vcard], { type: 'text/vcard;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
