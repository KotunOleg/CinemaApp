import { useState, useEffect } from 'react'
import { useAuth } from '../../context/AuthContext'
import { api } from '../../api/client'

const RULES = [
  { id: 'len',     label: 'At least 8 characters',       test: (p) => p.length >= 8 },
  { id: 'upper',   label: 'Uppercase letter (A–Z)',       test: (p) => /[A-Z]/.test(p) },
  { id: 'lower',   label: 'Lowercase letter (a–z)',       test: (p) => /[a-z]/.test(p) },
  { id: 'digit',   label: 'Number (0–9)',                 test: (p) => /\d/.test(p) },
  { id: 'special', label: 'Special character (!@#$…)',    test: (p) => /[^A-Za-z0-9]/.test(p) },
]

function getStrength(password) {
  if (!password) return null
  const passed = RULES.filter((r) => r.test(password)).length
  if (passed <= 2) return { level: 1, label: 'Weak',   color: 'danger' }
  if (passed === 3) return { level: 2, label: 'Fair',   color: 'warning' }
  if (passed === 4) return { level: 3, label: 'Good',   color: 'info' }
  return            { level: 4, label: 'Strong', color: 'success' }
}

function PasswordStrength({ password }) {
  const strength = getStrength(password)
  if (!strength) return null
  return (
    <div className="mt-2">
      <div className="d-flex gap-1 mb-1">
        {[1, 2, 3, 4].map((n) => (
          <div
            key={n}
            className="flex-grow-1 rounded"
            style={{
              height: 5,
              backgroundColor: n <= strength.level ? `var(--bs-${strength.color})` : '#dee2e6',
              transition: 'background-color 0.2s',
            }}
          />
        ))}
        <small className={`text-${strength.color} fw-semibold ms-1`} style={{ minWidth: 48 }}>
          {strength.label}
        </small>
      </div>
      <ul className="list-unstyled mb-0 mt-1" style={{ fontSize: 12 }}>
        {RULES.map((r) => {
          const ok = r.test(password)
          return (
            <li key={r.id} className={ok ? 'text-success' : 'text-muted'}>
              <i className={`bi bi-${ok ? 'check-circle-fill' : 'circle'} me-1`} />
              {r.label}
            </li>
          )
        })}
      </ul>
    </div>
  )
}


function formatTime(ts) {
  if (!ts) return '—'
  try { return new Date(ts).toLocaleString('uk-UA') } catch { return String(ts) }
}

function ShowtimesTab() {
  const [showtimes, setShowtimes] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get('/api/showtimes?limit=100')
      .then((data) => setShowtimes(data || []))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-center py-4"><div className="spinner-border text-warning" /></div>

  return (
    <>
      <h5 className="fw-bold mb-3">Upcoming Showtimes</h5>
      {showtimes.length === 0 ? (
        <p className="text-muted">No showtimes available.</p>
      ) : (
        <div className="table-responsive">
          <table className="table table-hover align-middle">
            <thead className="table-light">
              <tr>
                <th>Movie</th>
                <th>Cinema</th>
                <th>Start Time</th>
                <th>Duration</th>
                <th>Price</th>
              </tr>
            </thead>
            <tbody>
              {showtimes.map((s) => (
                <tr key={s.showtime_id}>
                  <td className="fw-semibold">{s.movie_title || '—'}</td>
                  <td>{s.cinema_name || '—'}</td>
                  <td>{formatTime(s.start_time)}</td>
                  <td>{s.duration} min</td>
                  <td className="text-success fw-semibold">{s.price} ₴</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}

function TicketsTab() {
  const [tickets, setTickets] = useState([])
  const [loading, setLoading] = useState(true)
  const [sub, setSub] = useState('all')

  useEffect(() => {
    api.get('/api/profile/tickets')
      .then((data) => setTickets(data || []))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="text-center py-4"><div className="spinner-border text-warning" /></div>

  const now = new Date()
  const filtered = sub === 'future'
    ? tickets.filter((t) => t.start_time && new Date(t.start_time) > now)
    : tickets

  return (
    <>
      <div className="d-flex align-items-center justify-content-between mb-3">
        <h5 className="fw-bold mb-0">My Tickets</h5>
        <div className="btn-group btn-group-sm">
          <button
            className={`btn ${sub === 'all' ? 'btn-warning' : 'btn-outline-secondary'}`}
            onClick={() => setSub('all')}
          >
            All ({tickets.length})
          </button>
          <button
            className={`btn ${sub === 'future' ? 'btn-warning' : 'btn-outline-secondary'}`}
            onClick={() => setSub('future')}
          >
            Upcoming ({tickets.filter((t) => t.start_time && new Date(t.start_time) > now).length})
          </button>
        </div>
      </div>

      {filtered.length === 0 ? (
        <p className="text-muted">No tickets found.</p>
      ) : (
        <div className="d-flex flex-column gap-3">
          {filtered.map((t) => {
            const isFuture = t.start_time && new Date(t.start_time) > now
            return (
              <div key={t.ticket_id} className="card border-0 shadow-sm">
                <div className="card-body d-flex justify-content-between align-items-center">
                  <div>
                    <div className="fw-bold">{t.movie_title}</div>
                    <div className="text-muted small">
                      <i className="bi bi-building me-1" />{t.cinema_name}
                      <span className="mx-2">·</span>
                      <i className="bi bi-clock me-1" />{formatTime(t.start_time)}
                    </div>
                    <div className="text-muted small">
                      <i className="bi bi-ticket-perforated me-1" />Seat: <strong>{t.seat_number}</strong>
                    </div>
                  </div>
                  <div className="text-end">
                    <span className={`badge ${t.payment_status === 'paid' ? 'bg-success' : t.payment_status === 'cancelled' ? 'bg-danger' : 'bg-warning text-dark'}`}>
                      {t.payment_status || 'pending'}
                    </span>
                    {isFuture && (
                      <div className="mt-1">
                        <span className="badge bg-info text-dark">Upcoming</span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </>
  )
}

function SettingsTab() {
  const { user, setUserFromToken } = useAuth()
  const [nameForm, setNameForm] = useState({ full_name: user?.full_name || '' })
  const [passForm, setPassForm] = useState({ password: '', confirm: '' })
  const [nameMsg, setNameMsg] = useState(null)
  const [passMsg, setPassMsg] = useState(null)
  const [nameLoading, setNameLoading] = useState(false)
  const [passLoading, setPassLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  const passStrength = getStrength(passForm.password)
  const passwordsMatch = passForm.password === passForm.confirm
  const canChangePass = passStrength && passStrength.level === 4 && passForm.confirm && passwordsMatch

  const handleName = async (e) => {
    e.preventDefault()
    setNameMsg(null)
    setNameLoading(true)
    try {
      await api.put('/api/profile', { full_name: nameForm.full_name })
      const updated = { ...user, full_name: nameForm.full_name }
      localStorage.setItem('user', JSON.stringify(updated))
      setUserFromToken(updated)
      setNameMsg({ type: 'success', text: 'Name updated!' })
    } catch (err) {
      setNameMsg({ type: 'danger', text: err.message })
    } finally {
      setNameLoading(false)
    }
  }

  const handlePass = async (e) => {
    e.preventDefault()
    setPassMsg(null)
    if (!canChangePass) return
    setPassLoading(true)
    try {
      await api.put('/api/profile', { password: passForm.password })
      setPassForm({ password: '', confirm: '' })
      setPassMsg({ type: 'success', text: 'Password updated!' })
    } catch (err) {
      setPassMsg({ type: 'danger', text: err.message })
    } finally {
      setPassLoading(false)
    }
  }

  return (
    <>
      <h5 className="fw-bold mb-4">Settings</h5>

      <div className="card border-0 shadow-sm mb-4">
        <div className="card-body">
          <h6 className="fw-semibold mb-3">Change Name</h6>
          {nameMsg && <div className={`alert alert-${nameMsg.type} py-2`}>{nameMsg.text}</div>}
          <form onSubmit={handleName}>
            <div className="mb-3">
              <label className="form-label">Full Name</label>
              <input
                type="text"
                className="form-control"
                value={nameForm.full_name}
                onChange={(e) => setNameForm({ full_name: e.target.value })}
                required
              />
            </div>
            <button type="submit" className="btn btn-warning" disabled={nameLoading}>
              {nameLoading ? 'Saving…' : 'Save Name'}
            </button>
          </form>
        </div>
      </div>

      <div className="card border-0 shadow-sm">
        <div className="card-body">
          <h6 className="fw-semibold mb-3">Change Password</h6>
          {passMsg && <div className={`alert alert-${passMsg.type} py-2`}>{passMsg.text}</div>}
          <form onSubmit={handlePass}>
            <div className="mb-3">
              <label className="form-label">New Password</label>
              <div className="input-group">
                <input
                  type={showPass ? 'text' : 'password'}
                  className={`form-control ${passForm.password && (passStrength && passStrength.level === 4 ? 'is-valid' : 'is-invalid')}`}
                  value={passForm.password}
                  onChange={(e) => setPassForm({ ...passForm, password: e.target.value })}
                  required
                />
                <button
                  type="button"
                  className="btn btn-outline-secondary"
                  onClick={() => setShowPass((v) => !v)}
                  tabIndex={-1}
                >
                  <i className={`bi bi-eye${showPass ? '-slash' : ''}`} />
                </button>
              </div>
              <div className="mt-2" style={{ fontSize: 13 }}>
                  {passStrength && (
                    <div className="d-flex gap-1 mb-1 align-items-center">
                      {[1, 2, 3, 4].map((n) => (
                        <div
                          key={n}
                          className="flex-grow-1 rounded"
                          style={{
                            height: 5,
                            backgroundColor: n <= passStrength.level ? `var(--bs-${passStrength.color})` : '#dee2e6',
                            transition: 'background-color 0.2s',
                          }}
                        />
                      ))}
                      <small className={`text-${passStrength.color} fw-semibold ms-1`} style={{ minWidth: 48 }}>
                        {passStrength.label}
                      </small>
                    </div>
                  )}
                  <ul className="list-unstyled mb-0 mt-1">
                    {RULES.map((r) => {
                      const ok = passForm.password && r.test(passForm.password)
                      return (
                        <li key={r.id} className={ok ? 'text-success' : 'text-muted'}>
                          <i className={`bi bi-${ok ? 'check-circle-fill' : 'circle'} me-1`} />
                          {r.label}
                        </li>
                      )
                    })}
                  </ul>
              </div>
            </div>
            <div className="mb-3">
              <label className="form-label">Confirm Password</label>
              <div className="input-group">
                <input
                  type={showConfirm ? 'text' : 'password'}
                  className={`form-control ${passForm.confirm && (passwordsMatch ? 'is-valid' : 'is-invalid')}`}
                  value={passForm.confirm}
                  onChange={(e) => setPassForm({ ...passForm, confirm: e.target.value })}
                  required
                />
                <button
                  type="button"
                  className="btn btn-outline-secondary"
                  onClick={() => setShowConfirm((v) => !v)}
                  tabIndex={-1}
                >
                  <i className={`bi bi-eye${showConfirm ? '-slash' : ''}`} />
                </button>
              </div>
              {passForm.confirm && !passwordsMatch && (
                <div className="text-danger mt-1" style={{ fontSize: 13 }}>
                  <i className="bi bi-x-circle me-1" />Passwords do not match
                </div>
              )}
              {passForm.confirm && passwordsMatch && (
                <div className="text-success mt-1" style={{ fontSize: 13 }}>
                  <i className="bi bi-check-circle me-1" />Passwords match
                </div>
              )}
            </div>
            <button type="submit" className="btn btn-warning" disabled={passLoading || !canChangePass}>
              {passLoading ? 'Saving…' : 'Change Password'}
            </button>
            {passForm.password && !(passStrength && passStrength.level === 4) && (
              <p className="text-danger mt-2 mb-0" style={{ fontSize: 13 }}>
                Password must meet all requirements.
              </p>
            )}
          </form>
        </div>
      </div>
    </>
  )
}

const TABS = [
  { key: 'showtimes', icon: 'calendar-event', label: 'Showtimes' },
  { key: 'tickets', icon: 'ticket-perforated', label: 'Tickets' },
  { key: 'settings', icon: 'gear', label: 'Settings' },
]

export default function ProfilePage() {
  const { user } = useAuth()
  const [tab, setTab] = useState('showtimes')

  return (
    <div className="container py-5">
      <div className="row g-4">
        {/* Sidebar */}
        <div className="col-md-3">
          <div className="card border-0 shadow-sm">
            <div className="card-body text-center py-4">
              <div
                className="rounded-circle bg-warning d-flex align-items-center justify-content-center mx-auto mb-2"
                style={{ width: 64, height: 64, fontSize: 28 }}
              >
                <i className="bi bi-person-fill text-dark" />
              </div>
              <div className="fw-bold">{user?.full_name}</div>
              <div className="text-muted small">{user?.email}</div>
            </div>
            <ul className="nav flex-column px-2 pb-3">
              {TABS.map(({ key, icon, label }) => (
                <li key={key} className="nav-item">
                  <button
                    className={`nav-link w-100 text-start ${tab === key ? 'active text-warning fw-semibold' : 'text-secondary'}`}
                    style={{ background: 'none', border: 'none' }}
                    onClick={() => setTab(key)}
                  >
                    <i className={`bi bi-${icon} me-2`} />
                    {label}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Content */}
        <div className="col-md-9">
          <div className="card border-0 shadow-sm">
            <div className="card-body p-4">
              {tab === 'showtimes' && <ShowtimesTab />}
              {tab === 'tickets' && <TicketsTab />}
              {tab === 'settings' && <SettingsTab />}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
