import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ email: '', password: '' })
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const user = await login(form.email, form.password)
      navigate(user.permission_level === 1 ? '/' : '/browse')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-vh-100 d-flex align-items-center justify-content-center bg-dark">
      <div className="card shadow-lg" style={{ width: 400 }}>
        <div className="card-body p-4">
          <div className="text-center mb-4">
            <i className="bi bi-camera-reels fs-1 text-warning" />
            <h4 className="mt-2 fw-bold">KotuKino</h4>
            <p className="text-muted">Sign in to continue</p>
          </div>

          <div className="alert alert-warning py-2 small mb-3">
            <i className="bi bi-shield-lock me-1" />
            <strong>Admin:</strong> admin@cinema.com / Admin_1234
          </div>

          {error && <div className="alert alert-danger py-2">{error}</div>}

          <form onSubmit={handleSubmit}>
            <div className="mb-3">
              <label className="form-label fw-semibold">Email</label>
              <input
                type="email"
                className="form-control"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
                autoFocus
              />
            </div>
            <div className="mb-4">
              <label className="form-label fw-semibold">Password</label>
              <div className="input-group">
                <input
                  type={showPass ? 'text' : 'password'}
                  className="form-control"
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
            </div>
            <button type="submit" className="btn btn-warning w-100 fw-bold" disabled={loading}>
              {loading ? 'Signing in…' : 'Sign In'}
            </button>
          </form>

          <div className="d-grid mt-3">
            <a href="/api/auth/google" className="btn btn-outline-secondary">
              <img src="https://www.gstatic.com/firebasejs/ui/2.0.0/images/auth/google.svg" alt="" width="18" className="me-2" />
              Sign in with Google
            </a>
          </div>

          <hr />
          <p className="text-center text-muted mb-0">
            Don't have an account?{' '}
            <Link to="/register" className="text-warning">Register</Link>
          </p>
        </div>
      </div>
    </div>
  )
}
