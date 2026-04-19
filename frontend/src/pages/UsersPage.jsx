import { useCallback } from 'react'
import { api } from '../api/client'
import EntityPage from '../components/EntityPage'

const columns = [
  { key: 'user_id', label: 'ID' },
  { key: 'full_name', label: 'Full Name' },
  { key: 'email', label: 'Email' },
  { key: 'phone', label: 'Phone', render: (v) => v?.String || '—' },
  { key: 'permission_level', label: 'Role', render: (v) => {
    const n = v?.Int32 ?? v
    return n === 1 ? <span className="badge bg-danger">Admin</span>
      : <span className="badge bg-secondary">User</span>
  }},
  { key: 'is_blocked', label: 'Blocked', render: (v) => {
    const b = v?.Bool ?? v
    return b ? <span className="badge bg-warning text-dark">Blocked</span>
      : <span className="badge bg-success">Active</span>
  }},
]

const fields = [
  { key: 'full_name', label: 'Full Name', required: true },
  { key: 'email', label: 'Email', type: 'email', required: true },
  { key: 'phone', label: 'Phone' },
  { key: 'password', label: 'Password (create only)', type: 'password' },
  { key: 'permission_level', label: 'Role', type: 'select', numericValue: true, options: [
    { value: 1, label: 'Admin' },
    { value: 2, label: 'User' },
  ]},
  { key: 'is_blocked', label: 'Blocked', type: 'checkbox' },
]

const defaultValues = { full_name: '', email: '', phone: '', password: '', permission_level: 2, is_blocked: false }

export default function UsersPage() {
  const fetchAll = useCallback(() => api.get('/api/users?limit=100'), [])
  const create = useCallback((data) => api.post('/api/users', {
    email: data.email,
    password: data.password,
    phone: data.phone || '',
    full_name: data.full_name,
    permission_level: Number(data.permission_level) || 2,
  }), [])
  const update = useCallback((id, data) => api.put(`/api/users/${id}`, {
    email: data.email,
    phone: data.phone || '',
    full_name: data.full_name,
    is_blocked: !!data.is_blocked,
    permission_level: Number(data.permission_level) || 2,
  }), [])
  const remove = useCallback((id) => api.delete(`/api/users/${id}`), [])

  return (
    <EntityPage
      title="Users"
      columns={columns}
      idKey="user_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
    />
  )
}
