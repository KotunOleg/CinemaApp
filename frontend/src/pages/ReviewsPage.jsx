import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import EntityPage from '../components/EntityPage'

const columns = [
  { key: 'review_id', label: 'ID' },
  { key: 'movie_title', label: 'Movie' },
  { key: 'user_name', label: 'User' },
  { key: 'rating', label: 'Rating', render: (v) => v ? `${v}/10` : '—' },
  { key: 'content', label: 'Review', render: (v) => {
    const s = v || ''
    return s.length > 80 ? s.slice(0, 80) + '…' : s || '—'
  }},
]

const defaultValues = { user_id: '', movie_id: '', rating: '', content: '' }

export default function ReviewsPage() {
  const [userOptions, setUserOptions] = useState([])
  const [movieOptions, setMovieOptions] = useState([])

  useEffect(() => {
    api.get('/api/users?limit=200').then((data) =>
      setUserOptions((data || []).map((u) => ({ value: u.user_id, label: `${u.full_name} (${u.email})` })))
    )
    api.get('/api/movies?limit=200').then((data) =>
      setMovieOptions((data || []).map((m) => ({ value: m.movie_id, label: m.title })))
    )
  }, [])

  const fields = [
    { key: 'user_id', label: 'User', type: 'select', numericValue: true, required: true, options: userOptions },
    { key: 'movie_id', label: 'Movie', type: 'select', numericValue: true, required: true, options: movieOptions },
    { key: 'rating', label: 'Rating (1-10)', type: 'number', min: 1, max: 10 },
    { key: 'content', label: 'Review', type: 'textarea' },
  ]

  const fetchAll = useCallback(() => api.get('/api/reviews?limit=100'), [])
  const create = useCallback((data) => api.post('/api/reviews', {
    ...data,
    user_id: Number(data.user_id),
    movie_id: Number(data.movie_id),
    rating: Number(data.rating) || 0,
  }), [])
  const update = useCallback((id, data) => api.put(`/api/reviews/${id}`, {
    rating: Number(data.rating) || 0,
    content: data.content,
  }), [])
  const remove = useCallback((id) => api.delete(`/api/reviews/${id}`), [])

  return (
    <EntityPage
      title="Reviews"
      columns={columns}
      idKey="review_id"
      fetchAll={fetchAll}
      create={create}
      update={update}
      remove={remove}
      fields={fields}
      defaultValues={defaultValues}
    />
  )
}
