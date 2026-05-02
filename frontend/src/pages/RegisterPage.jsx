import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

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

export default function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ email: '', password: '', confirm: '', full_name: '', phone: '' })
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  const strength = getStrength(form.password)
  const passwordsMatch = form.password === form.confirm
  const canSubmit = strength && strength.level === 4 && form.confirm && passwordsMatch

  async function handleSubmit(e) {
    e.preventDefault()
    if (!canSubmit) return
    setError(null)
    setLoading(true)
    try {
      await register(form.email, form.password, form.full_name, form.phone)
      navigate('/browse')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-vh-100 d-flex align-items-center justify-content-center bg-dark">
      <div className="card shadow-lg" style={{ width: 420 }}>
        <div className="card-body p-4">
          <div className="text-center mb-4">
            <i className="bi bi-camera-reels fs-1 text-warning" />
            <h4 className="mt-2 fw-bold">Create Account</h4>
            <p className="text-muted">Join KotuKino today</p>
          </div>

          {error && <div className="alert alert-danger py-2">{error}</div>}

          <form onSubmit={handleSubmit}>
            <div className="mb-3">
              <label className="form-label fw-semibold">Full Name</label>
              <input
                type="text"
                className="form-control"
                value={form.full_name}
                onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                required
                autoFocus
              />
            </div>
            <div className="mb-3">
              <label className="form-label fw-semibold">Email</label>
              <input
                type="email"
                className="form-control"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
              />
            </div>
            <div className="mb-3">
              <label className="form-label fw-semibold">Phone (optional)</label>
              <input
                type="tel"
                className="form-control"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
            </div>
            <div className="mb-3">
              <label className="form-label fw-semibold">Password</label>
              <div className="input-group">
                <input
                  type={showPass ? 'text' : 'password'}
                  className={`form-control ${form.password && (strength && strength.level === 4 ? 'is-valid' : 'is-invalid')}`}
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
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
              <PasswordStrength password={form.password} />
            </div>
            <div className="mb-4">
              <label className="form-label fw-semibold">Confirm Password</label>
              <div className="input-group">
                <input
                  type={showConfirm ? 'text' : 'password'}
                  className={`form-control ${form.confirm && (passwordsMatch ? 'is-valid' : 'is-invalid')}`}
                  value={form.confirm}
                  onChange={(e) => setForm({ ...form, confirm: e.target.value })}
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
              {form.confirm && !passwordsMatch && (
                <div className="text-danger mt-1" style={{ fontSize: 13 }}>
                  <i className="bi bi-x-circle me-1" />Passwords do not match
                </div>
              )}
              {form.confirm && passwordsMatch && (
                <div className="text-success mt-1" style={{ fontSize: 13 }}>
                  <i className="bi bi-check-circle me-1" />Passwords match
                </div>
              )}
            </div>
            <button
              type="submit"
              className="btn btn-warning w-100 fw-bold"
              disabled={loading || !canSubmit}
            >
              {loading ? 'Registering…' : 'Register'}
            </button>
            {form.password && !(strength && strength.level === 4) && (
              <p className="text-danger text-center mt-2 mb-0" style={{ fontSize: 13 }}>
                Password must meet all requirements.
              </p>
            )}
          </form>

          <hr />
          <p className="text-center text-muted mb-0">
            Already have an account?{' '}
            <Link to="/login" className="text-warning">Sign In</Link>
          </p>
        </div>
      </div>
    </div>
  )
}
