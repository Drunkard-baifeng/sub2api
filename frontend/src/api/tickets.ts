import { apiClient } from './client'

export type TicketStatus = 'pending' | 'processing' | 'waiting_user' | 'resolved'
export interface Ticket {
  id: number; user_id: number; email?: string; subject: string; category: string
  status: TicketStatus; last_message_id: number; unread: boolean; created_at: string; updated_at: string; expires_at?: string
}
export interface TicketAttachment { id: number; mime: string; size: number }
export interface TicketMessage {
  id: number; is_admin: boolean; body: string; status_event: string; created_at: string
  attachments: TicketAttachment[]
}
export interface TicketDetail { ticket: Ticket; messages: TicketMessage[]; has_more: boolean }
export const ticketBase = (admin = false) => admin ? '/admin/tickets' : '/tickets'
export const ticketsAPI = {
  async list(admin: boolean, params: { page: number; status: string; unread: boolean }) {
    return (await apiClient.get<{ items: Ticket[]; total: number; retention_days?: number }>(ticketBase(admin), { params })).data
  },
  async unread(admin: boolean) {
    return (await apiClient.get<{ count: number }>(ticketBase(admin) + '/unread')).data.count
  },
  async detail(admin: boolean, id: number, before = 0) {
    return (await apiClient.get<TicketDetail>(ticketBase(admin) + '/' + id, { params: { before } })).data
  },
  async create(form: FormData) {
    return (await apiClient.post<{ id: number }>('/tickets', form, { headers: { 'Content-Type': undefined } })).data.id
  },
  async reply(admin: boolean, id: number, form: FormData) {
    await apiClient.post(ticketBase(admin) + '/' + id + '/messages', form, { headers: { 'Content-Type': undefined } })
  },
  async status(admin: boolean, id: number, status: TicketStatus) {
    await apiClient.patch(ticketBase(admin) + '/' + id + '/status', { status })
  },
  async remove(id: number) {
    await apiClient.delete('/admin/tickets/' + id)
  },
  async read(admin: boolean, id: number, message_id: number) {
    await apiClient.post(ticketBase(admin) + '/' + id + '/read', { message_id })
  },
  async attachment(admin: boolean, id: number, attachment: number) {
    return (await apiClient.get<Blob>(ticketBase(admin) + '/' + id + '/attachments/' + attachment, { responseType: 'blob' })).data
  },
}
export function notifyTicketsChanged() { window.dispatchEvent(new Event('tickets-changed')) }
