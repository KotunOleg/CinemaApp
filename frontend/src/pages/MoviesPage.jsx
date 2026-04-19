import { useCallback, useRef } from 'react'
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
  const importRef = useRef(null)

  function handleExport() {
    window.location.href = '/api/movies/export'
  }

  async function handleImport(e) {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''
    const form = new FormData()
    form.append('file', file)
    try {
      const res = await fetch('/api/movies/import', { method: 'POST', body: form })
      const data = await res.json()
      if (!res.ok) throw new Error(data)
      alert(`Imported ${data.imported} movie(s) successfully.`)
      window.location.reload()
    } catch (err) {
      alert('Import failed: ' + err.message)
    }
  }

  const extraActions = (
    <div className="d-flex gap-2">
      <button className="btn btn-outline-success btn-sm" onClick={handleExport}>
        <i className="bi bi-file-earmark-excel me-1" />
        Export Excel
      </button>
      <button className="btn btn-outline-primary btn-sm" onClick={() => importRef.current?.click()}>
        <i className="bi bi-upload me-1" />
        Import Excel
      </button>
      <input ref={importRef} type="file" accept=".xlsx" className="d-none" onChange={handleImport} />
    </div>
  )

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
      extraActions={extraActions}
    />
  )
}
