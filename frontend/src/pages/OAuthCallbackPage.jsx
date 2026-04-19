import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { api } from '../api/client'

export default function OAuthCallbackPage() {
  const navigate = useNavigate()
  const { setUserFromToken } = useAuth()

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    if (!token) {
      navigate('/login')
      return
    }
    localStorage.setItem('token', token)
    api.get('/api/auth/me')
      .then((user) => {
        localStorage.setItem('user', JSON.stringify(user))
        setUserFromToken(user)
        navigate(user.permission_level === 1 ? '/' : '/browse')
      })
      .catch(() => navigate('/login'))
  }, [])

  return (
    <div className="min-vh-100 d-flex align-items-center justify-content-center bg-dark">
      <div className="text-white text-center">
        <div className="spinner-border text-warning mb-3" />
        <p>Signing you in…</p>
      </div>
    </div>
  )
}
