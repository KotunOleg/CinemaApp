import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../../api/client'

export default function BrowsePage() {
  const [movies, setMovies] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get('/api/movies?limit=100')
      .then(setMovies)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return (
    <div className="text-center py-5">
      <div className="spinner-border text-warning" />
    </div>
  )

  return (
    <div className="container py-5">
      <h2 className="fw-bold mb-4">Now Showing</h2>
      {movies.length === 0 && (
        <p className="text-muted">No movies available.</p>
      )}
      <div className="row g-4">
        {movies.map((m) => (
          <div className="col-sm-6 col-md-4 col-lg-3" key={m.movie_id}>
            <Link to={`/browse/movies/${m.movie_id}`} className="text-decoration-none">
              <div className="card h-100 shadow-sm border-0 hover-shadow">
                <div
                  className="card-img-top d-flex align-items-center justify-content-center bg-dark text-warning"
                  style={{ height: 180, fontSize: 48 }}
                >
                  <i className="bi bi-film" />
                </div>
                <div className="card-body">
                  <h6 className="card-title fw-bold text-dark">{m.title}</h6>
                  <p className="text-muted small mb-1">
                    {(m.genre || []).join(', ') || '—'}
                  </p>
                  <p className="text-muted small" style={{ fontSize: 12 }}>
                    {(m.description || '').slice(0, 80)}
                    {(m.description || '').length > 80 ? '…' : ''}
                  </p>
                </div>
                <div className="card-footer bg-white border-0">
                  <span className="btn btn-outline-warning btn-sm w-100">View Showtimes</span>
                </div>
              </div>
            </Link>
          </div>
        ))}
      </div>
    </div>
  )
}
