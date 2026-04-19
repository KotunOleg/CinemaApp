import { useCallback } from 'react'
import { api } from '../api/client'
import EntityPage from '../components/EntityPage'

const columns = [
  { key: 'movie_id', label: 'ID' },
  { key: 'title', label: 'Title' },
  { key: 'genre', label: 'Genre', render: (v) => (v || []).join(', ') || '—' },
  { key: 'description', label: 'Description', render: (v) => {
    const s = v || ''
    return s.length > 60 ? s.slice(0, 60) + '…' : s || '—'
  }},
  { key: 'trailer_url', label: 'Trailer', render: (v) => v
    ? <a href={v} target="_blank" rel="noreferrer">Link</a>
    : '—'
  },
]

const fields = [
  { key: 'title', label: 'Title', required: true },
  { key: 'description', label: 'Description', type: 'textarea' },
  { key: 'genre', label: 'Genre (comma-separated)', placeholder: 'Action, Drama' },
  { key: 'trailer_url', label: 'Trailer URL' },
]

const defaultValues = { title: '', description: '', genre: '', trailer_url: '' }

function prepareData(data) {
  return {
    ...data,
    genre: typeof data.genre === 'string'
      ? data.genre.split(',').map((s) => s.trim()).filter(Boolean)
      : data.genre || [],
  }
}

export default function MoviesPage() {
  const fetchAll = useCallback(() => api.get('/api/movies?limit=100'), [])
  const create = useCallback((data) => api.post('/api/movies', prepareData(data)), [])
  const update = useCallback((id, data) => api.put(`/api/movies/${id}`, prepareData(data)), [])
  const remove = useCallback((id) => api.delete(`/api/movies/${id}`), [])

  return (
    <EntityPage
      title="Movies"
      columns={columns}
      idKey="movie_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
    />
  )
}
