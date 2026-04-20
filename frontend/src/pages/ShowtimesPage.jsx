import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import EntityPage from '../components/EntityPage'

const fmt = (ts) => {
  if (!ts) return '—'
  try { return new Date(ts?.Time || ts).toLocaleString() } catch { return String(ts) }
}

const columns = [
  { key: 'showtime_id', label: 'ID' },
  { key: 'movie_title', label: 'Movie' },
  { key: 'cinema_name', label: 'Cinema' },
  { key: 'start_time', label: 'Start Time', render: (v) => fmt(v) },
  { key: 'price', label: 'Price' },
  { key: 'duration', label: 'Duration (min)' },
]

const defaultValues = { movie_id: '', cinema_id: '', start_time: '', price: '', duration: '' }

export default function ShowtimesPage() {
  const [movieOptions, setMovieOptions] = useState([])
  const [cinemaOptions, setCinemaOptions] = useState([])

  useEffect(() => {
    api.get('/api/movies?limit=200').then((data) =>
      setMovieOptions((data || []).map((m) => ({ value: m.movie_id, label: m.title })))
    )
    api.get('/api/cinemas?limit=200').then((data) =>
      setCinemaOptions((data || []).map((c) => ({ value: c.cinema_id, label: c.name })))
    )
  }, [])

  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  tomorrow.setHours(0, 0, 0, 0)
  const pad = (n) => String(n).padStart(2, '0')
  const tomorrowMin = `${tomorrow.getFullYear()}-${pad(tomorrow.getMonth() + 1)}-${pad(tomorrow.getDate())}T00:00`

  const fields = [
    { key: 'movie_id', label: 'Movie', type: 'select', numericValue: true, required: true, options: movieOptions },
    { key: 'cinema_id', label: 'Cinema', type: 'select', numericValue: true, required: true, options: cinemaOptions },
    { key: 'start_time', label: 'Start Time', type: 'datetime-local', required: true, min: tomorrowMin },
    { key: 'price', label: 'Price', type: 'number', step: '0.01', required: true },
    { key: 'duration', label: 'Duration (minutes)', type: 'number', required: true, min: 10 },
  ]

  const fetchAll = useCallback(() => api.get('/api/showtimes?limit=100'), [])
  const toISO = (data) => ({
    ...data,
    movie_id: Number(data.movie_id),
    cinema_id: Number(data.cinema_id),
    price: Number(data.price),
    duration: Number(data.duration),
    start_time: data.start_time ? new Date(data.start_time).toISOString() : '',
  })

  const create = useCallback((data) => api.post('/api/showtimes', toISO(data)), [])
  const update = useCallback((id, data) => api.put(`/api/showtimes/${id}`, toISO(data)), [])
  const remove = useCallback((id) => api.delete(`/api/showtimes/${id}`), [])

  return (
    <EntityPage
      title="Showtimes"
      columns={columns}
      idKey="showtime_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
    />
  )
}
