import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../../api/client'
import { useAuth } from '../../context/AuthContext'

function formatTime(ts) {
  if (!ts) return '—'
  try { return new Date(ts?.Time || ts).toLocaleString('uk-UA') } catch { return String(ts) }
}

function Stars({ rating, max = 10 }) {
  const n = Number(rating) || 0
  return (
    <span>
      {Array.from({ length: max }, (_, i) => (
        <i
          key={i}
          className={`bi bi-star${i < n ? '-fill' : ''} text-warning`}
          style={{ fontSize: 14 }}
        />
      ))}
      <span className="ms-1 text-muted small">{n}/{max}</span>
    </span>
  )
}

function StarPicker({ value, onChange }) {
  const [hover, setHover] = useState(0)
  return (
    <div className="d-flex gap-1">
      {Array.from({ length: 10 }, (_, i) => i + 1).map((n) => (
        <i
          key={n}
          className={`bi bi-star${n <= (hover || value) ? '-fill' : ''} text-warning`}
          style={{ fontSize: 22, cursor: 'pointer' }}
          onMouseEnter={() => setHover(n)}
          onMouseLeave={() => setHover(0)}
          onClick={() => onChange(n)}
        />
      ))}
    </div>
  )
}

export default function MoviePage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [movie, setMovie] = useState(null)
  const [showtimes, setShowtimes] = useState([])
  const [reviews, setReviews] = useState([])
  const [loading, setLoading] = useState(true)

  // Booking
  const [booking, setBooking] = useState(null)
  const [bookForm, setBookForm] = useState({ seat_number: '' })
  const [bookSubmitting, setBookSubmitting] = useState(false)
  const [bookSuccess, setBookSuccess] = useState(null)
  const [bookError, setBookError] = useState(null)

  // Review
  const [reviewForm, setReviewForm] = useState({ rating: 0, content: '' })
  const [reviewSubmitting, setReviewSubmitting] = useState(false)
  const [reviewError, setReviewError] = useState(null)

  const loadReviews = () =>
    api.get(`/api/movies/${id}/reviews`).then((data) => setReviews(data || []))

  useEffect(() => {
    Promise.all([
      api.get(`/api/movies/${id}`),
      api.get('/api/showtimes?limit=100'),
      api.get(`/api/movies/${id}/reviews`),
    ]).then(([mov, all, revs]) => {
      setMovie(mov)
      setShowtimes((all || []).filter((s) => s.movie_id === Number(id)))
      setReviews(revs || [])
    }).finally(() => setLoading(false))
  }, [id])

  const handleBook = async (e) => {
    e.preventDefault()
    setBookSubmitting(true)
    setBookError(null)
    try {
      await api.post('/api/tickets', {
        showtime_id: booking.showtime_id,
        user_id: user?.user_id,
        seat_number: bookForm.seat_number,
        payment_status: 'pending',
      })
      setBookSuccess(`Ticket booked! Seat ${bookForm.seat_number} — ${formatTime(booking.start_time)}`)
      setBooking(null)
      setBookForm({ seat_number: '' })
    } catch (e) {
      setBookError(e.message)
    } finally {
      setBookSubmitting(false)
    }
  }

  const handleReview = async (e) => {
    e.preventDefault()
    setReviewSubmitting(true)
    setReviewError(null)
    try {
      await api.post('/api/reviews', {
        user_id: user?.user_id,
        movie_id: Number(id),
        rating: reviewForm.rating,
        content: reviewForm.content,
      })
      setReviewForm({ rating: 0, content: '' })
      await loadReviews()
    } catch (e) {
      setReviewError(e.message)
    } finally {
      setReviewSubmitting(false)
    }
  }

  if (loading) return <div className="text-center py-5"><div className="spinner-border text-warning" /></div>
  if (!movie) return <div className="container py-5 text-muted">Movie not found.</div>

  const avgRating = reviews.length
    ? (reviews.reduce((s, r) => s + (Number(r.rating) || 0), 0) / reviews.length).toFixed(1)
    : null

  return (
    <div className="container py-5">
      <button className="btn btn-link text-muted ps-0 mb-3" onClick={() => navigate('/browse')}>
        <i className="bi bi-arrow-left me-1" /> Back to movies
      </button>

      {/* Movie header */}
      <div className="row g-4 mb-5">
        <div className="col-auto">
          <div
            className="d-flex align-items-center justify-content-center bg-dark text-warning rounded"
            style={{ width: 140, height: 200, fontSize: 64 }}
          >
            <i className="bi bi-film" />
          </div>
        </div>
        <div className="col">
          <h2 className="fw-bold">{movie.title}</h2>
          <p className="text-muted mb-1">{(movie.genre || []).join(' · ') || '—'}</p>
          {avgRating && (
            <p className="mb-2">
              <Stars rating={Math.round(avgRating)} />
              <span className="ms-2 text-muted small">{avgRating} / 10 ({reviews.length} reviews)</span>
            </p>
          )}
          <p className="mb-3">{movie.description || ''}</p>
          {movie.trailer_url && (
            <a href={movie.trailer_url} target="_blank" rel="noreferrer" className="btn btn-outline-danger btn-sm">
              <i className="bi bi-youtube me-1" /> Watch Trailer
            </a>
          )}
        </div>
      </div>

      {bookSuccess && (
        <div className="alert alert-success alert-dismissible">
          <i className="bi bi-check-circle me-2" />{bookSuccess}
          <button className="btn-close" onClick={() => setBookSuccess(null)} />
        </div>
      )}

      {/* Showtimes */}
      <h4 className="fw-bold mb-3">Showtimes</h4>
      {showtimes.length === 0 ? (
        <p className="text-muted mb-5">No showtimes scheduled.</p>
      ) : (
        <div className="row g-3 mb-5">
          {showtimes.map((s) => (
            <div className="col-md-6 col-lg-4" key={s.showtime_id}>
              <div className="card border-0 shadow-sm h-100">
                <div className="card-body">
                  <h6 className="fw-bold mb-1">{s.cinema_name}</h6>
                  <p className="text-muted small mb-1">
                    <i className="bi bi-clock me-1" />{formatTime(s.start_time)}
                  </p>
                  <p className="text-muted small mb-2">
                    <i className="bi bi-hourglass me-1" />{s.duration} min
                  </p>
                  <p className="fw-bold text-success mb-3">{s.price} ₴</p>
                  <button
                    className="btn btn-warning btn-sm w-100"
                    onClick={() => { setBooking(s); setBookError(null) }}
                  >
                    <i className="bi bi-ticket-perforated me-1" /> Book Ticket
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Reviews */}
      <h4 className="fw-bold mb-3">
        Reviews
        {reviews.length > 0 && <span className="text-muted fw-normal fs-6 ms-2">({reviews.length})</span>}
      </h4>

      {reviews.length === 0 ? (
        <p className="text-muted">No reviews yet. Be the first!</p>
      ) : (
        <div className="d-flex flex-column gap-3 mb-4">
          {reviews.map((r) => (
            <div className="card border-0 shadow-sm" key={r.review_id}>
              <div className="card-body">
                <div className="d-flex justify-content-between align-items-start mb-1">
                  <span className="fw-semibold">{r.user_name}</span>
                  <Stars rating={r.rating} />
                </div>
                <p className="mb-0 text-muted">{r.content || ''}</p>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Leave a review */}
      <div className="card border-0 shadow-sm mt-3">
        <div className="card-body">
          <h6 className="fw-bold mb-3">Leave a Review</h6>
          <form onSubmit={handleReview}>
            {reviewError && <div className="alert alert-danger py-2">{reviewError}</div>}
            <div className="mb-3">
              <label className="form-label fw-semibold">Rating</label>
              <div>
                <StarPicker
                  value={reviewForm.rating}
                  onChange={(n) => setReviewForm({ ...reviewForm, rating: n })}
                />
              </div>
            </div>
            <div className="mb-3">
              <label className="form-label fw-semibold">Review</label>
              <textarea
                className="form-control"
                rows={3}
                placeholder="Share your thoughts…"
                value={reviewForm.content}
                onChange={(e) => setReviewForm({ ...reviewForm, content: e.target.value })}
              />
            </div>
            <button
              type="submit"
              className="btn btn-warning"
              disabled={reviewSubmitting || !reviewForm.rating}
            >
              {reviewSubmitting ? 'Submitting…' : 'Submit Review'}
            </button>
          </form>
        </div>
      </div>

      {/* Booking modal */}
      {booking && (
        <div className="modal show d-block" tabIndex="-1" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div className="modal-dialog modal-dialog-centered">
            <div className="modal-content">
              <form onSubmit={handleBook}>
                <div className="modal-header">
                  <h5 className="modal-title">Book Ticket</h5>
                  <button type="button" className="btn-close" onClick={() => setBooking(null)} />
                </div>
                <div className="modal-body">
                  <p className="text-muted mb-3">
                    <strong>{movie.title}</strong> · {booking.cinema_name}<br />
                    <i className="bi bi-clock me-1" />{formatTime(booking.start_time)}
                  </p>
                  {bookError && <div className="alert alert-danger py-2">{bookError}</div>}
                  <div className="mb-3">
                    <label className="form-label fw-semibold">Seat Number</label>
                    <input
                      type="text"
                      className="form-control"
                      placeholder="e.g. A12"
                      value={bookForm.seat_number}
                      onChange={(e) => setBookForm({ ...bookForm, seat_number: e.target.value })}
                      required
                    />
                  </div>
                  <p className="fw-bold text-success">Total: {booking.price} ₴</p>
                </div>
                <div className="modal-footer">
                  <button type="button" className="btn btn-secondary" onClick={() => setBooking(null)}>Cancel</button>
                  <button type="submit" className="btn btn-warning" disabled={bookSubmitting}>
                    {bookSubmitting ? 'Booking…' : 'Confirm Booking'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
