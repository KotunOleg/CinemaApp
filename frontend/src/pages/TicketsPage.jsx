import { useCallback, useEffect, useState } from 'react'
import { api, downloadFile } from '../api/client'
import EntityPage from '../components/EntityPage'

const statusBadge = (status) => {
  const s = status || ''
  const cls = s === 'paid' ? 'success' : s === 'cancelled' ? 'danger' : 'warning'
  return <span className={`badge bg-${cls}`}>{s || 'pending'}</span>
}

const fmt = (ts) => {
  if (!ts) return '—'
  try { return new Date(ts?.Time || ts).toLocaleString() } catch { return String(ts) }
}

const columns = [
  { key: 'ticket_id', label: 'Ticket ID', render: (v) => (
    <code style={{ fontSize: 11 }}>{typeof v === 'string' ? v : String(v)}</code>
  )},
  { key: 'movie_title', label: 'Movie' },
  { key: 'cinema_name', label: 'Cinema' },
  { key: 'start_time', label: 'Showtime', render: (v) => fmt(v) },
  { key: 'user_name', label: 'User' },
  { key: 'seat_number', label: 'Seat' },
  { key: 'payment_status', label: 'Status', render: (v) => statusBadge(v) },
]

const defaultValues = { showtime_id: '', user_id: '', seat_number: '', payment_status: 'pending' }

export default function TicketsPage() {
  const [showtimeOptions, setShowtimeOptions] = useState([])
  const [userOptions, setUserOptions] = useState([])

  useEffect(() => {
    api.get('/api/showtimes?limit=200').then((data) =>
      setShowtimeOptions((data || []).map((s) => ({
        value: s.showtime_id,
        label: `${s.movie_title} — ${s.cinema_name} (${fmt(s.start_time)})`,
      })))
    )
    api.get('/api/users?limit=200').then((data) =>
      setUserOptions((data || []).map((u) => ({ value: u.user_id, label: `${u.full_name} (${u.email})` })))
    )
  }, [])

  const fields = [
    { key: 'showtime_id', label: 'Showtime', type: 'select', numericValue: true, required: true, options: showtimeOptions },
    { key: 'user_id', label: 'User', type: 'select', numericValue: true, required: true, options: userOptions },
    { key: 'seat_number', label: 'Seat Number', required: true, placeholder: 'A12' },
    { key: 'payment_status', label: 'Payment Status', type: 'select', options: [
      { value: 'pending', label: 'Pending' },
      { value: 'paid', label: 'Paid' },
      { value: 'cancelled', label: 'Cancelled' },
    ]},
  ]

  const fetchAll = useCallback(() => api.get('/api/tickets?limit=100'), [])
  const create = useCallback((data) => api.post('/api/tickets', {
    ...data,
    showtime_id: Number(data.showtime_id),
    user_id: Number(data.user_id),
  }), [])
  const update = useCallback((id, data) => api.patch(`/api/tickets/${id}`, {
    payment_status: data.payment_status,
  }), [])
  const remove = useCallback((id) => api.delete(`/api/tickets/${id}`), [])

  return (
    <EntityPage
      title="Tickets"
      columns={columns}
      idKey="ticket_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
      extraActions={
        <button className="btn btn-success btn-sm" onClick={() => downloadFile('/api/tickets/export', 'tickets.xlsx')}>
          <i className="bi bi-file-earmark-excel me-1" />Export Excel
        </button>
      }
    />
  )
}
