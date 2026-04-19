import { useCallback } from 'react'
import { api } from '../api/client'
import EntityPage from '../components/EntityPage'

const columns = [
  { key: 'cinema_id', label: 'ID' },
  { key: 'name', label: 'Name' },
  { key: 'address', label: 'Address' },
  { key: 'location_coordinates', label: 'Coordinates', render: (v) => v || '—' },
]

const fields = [
  { key: 'name', label: 'Name', required: true },
  { key: 'address', label: 'Address', type: 'textarea', required: true },
  { key: 'location_coordinates', label: 'Coordinates (lat,lng)', placeholder: '50.45,30.52' },
]

const defaultValues = { name: '', address: '', location_coordinates: '' }

export default function CinemasPage() {
  const fetchAll = useCallback(() => api.get('/api/cinemas?limit=100'), [])
  const create = useCallback((data) => api.post('/api/cinemas', data), [])
  const update = useCallback((id, data) => api.put(`/api/cinemas/${id}`, data), [])
  const remove = useCallback((id) => api.delete(`/api/cinemas/${id}`), [])

  return (
    <EntityPage
      title="Cinemas"
      columns={columns}
      idKey="cinema_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
    />
  )
}
