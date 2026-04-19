import { Routes, Route, NavLink, Link, useLocation, Navigate } from 'react-router-dom'
import CinemasPage from './pages/CinemasPage'
import MoviesPage from './pages/MoviesPage'
import ShowtimesPage from './pages/ShowtimesPage'
import TicketsPage from './pages/TicketsPage'
import UsersPage from './pages/UsersPage'
import ReviewsPage from './pages/ReviewsPage'
import StatisticsPage from './pages/StatisticsPage'
import BrowsePage from './pages/client/BrowsePage'
import MoviePage from './pages/client/MoviePage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import OAuthCallbackPage from './pages/OAuthCallbackPage'
import { useAuth } from './context/AuthContext'

const adminNavItems = [
  { to: '/cinemas', icon: 'building', label: 'Cinemas' },
  { to: '/movies', icon: 'film', label: 'Movies' },
  { to: '/showtimes', icon: 'calendar-event', label: 'Showtimes' },
  { to: '/tickets', icon: 'ticket-perforated', label: 'Tickets' },
  { to: '/reviews', icon: 'star', label: 'Reviews' },
  { to: '/users', icon: 'people', label: 'Users' },
  { to: '/statistics', icon: 'bar-chart-fill', label: 'Statistics' },
]

function AdminLayout() {
  const { user, logout } = useAuth()

  if (!user || user.permission_level !== 1) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="d-flex vh-100">
      <nav className="d-flex flex-column p-3 bg-dark text-white" style={{ width: 220, minWidth: 220 }}>
        <div className="mb-4 px-2">
          <h5 className="fw-bold text-warning mb-0">
            <i className="bi bi-camera-reels me-2" />
            KotuKino
          </h5>
          <small className="text-secondary">Admin Panel</small>
        </div>
        <ul className="nav nav-pills flex-column gap-1 flex-grow-1">
          {adminNavItems.map(({ to, icon, label }) => (
            <li key={to} className="nav-item">
              <NavLink
                to={to}
                className={({ isActive }) =>
                  `nav-link text-white ${isActive ? 'bg-primary' : ''}`
                }
              >
                <i className={`bi bi-${icon} me-2`} />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
        <div className="border-top border-secondary pt-3 mt-2">
          <Link to="/browse" className="nav-link text-warning mb-2">
            <i className="bi bi-eye me-2" />
            Client View
          </Link>
          <div className="px-2">
            <small className="text-secondary d-block text-truncate">{user.email}</small>
            <button className="btn btn-sm btn-outline-danger mt-1 w-100" onClick={logout}>
              <i className="bi bi-box-arrow-right me-1" />
              Logout
            </button>
          </div>
        </div>
      </nav>

      <main className="flex-grow-1 overflow-auto bg-light">
        <Routes>
          <Route path="/" element={<CinemasPage />} />
          <Route path="/cinemas" element={<CinemasPage />} />
          <Route path="/movies" element={<MoviesPage />} />
          <Route path="/showtimes" element={<ShowtimesPage />} />
          <Route path="/tickets" element={<TicketsPage />} />
          <Route path="/reviews" element={<ReviewsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/statistics" element={<StatisticsPage />} />
        </Routes>
      </main>
    </div>
  )
}

function ClientLayout() {
  const { user, logout } = useAuth()

  return (
    <div className="min-vh-100 bg-light">
      <nav className="navbar navbar-dark bg-dark px-4">
        <Link to="/browse" className="navbar-brand fw-bold text-warning">
          <i className="bi bi-camera-reels me-2" />
          KotuKino
        </Link>
        <div className="d-flex align-items-center gap-2">
          {user ? (
            <>
              {user.permission_level === 1 && (
                <Link to="/" className="btn btn-outline-secondary btn-sm">
                  <i className="bi bi-gear me-1" />
                  Admin
                </Link>
              )}
              <span className="text-secondary small">
                <i className="bi bi-person-circle me-1" />{user.full_name}
              </span>
              <button className="btn btn-outline-danger btn-sm" onClick={logout}>
                <i className="bi bi-box-arrow-right me-1" />
                Logout
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="btn btn-outline-warning btn-sm">Sign In</Link>
              <Link to="/register" className="btn btn-warning btn-sm">Register</Link>
            </>
          )}
        </div>
      </nav>
      <Routes>
        <Route path="/browse" element={<BrowsePage />} />
        <Route path="/browse/movies/:id" element={<MoviePage />} />
      </Routes>
    </div>
  )
}

export default function App() {
  const location = useLocation()
  const isBrowse = location.pathname.startsWith('/browse')
  const isAuth = ['/login', '/register', '/oauth'].includes(location.pathname)

  if (isAuth) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/oauth" element={<OAuthCallbackPage />} />
      </Routes>
    )
  }

  return isBrowse ? <ClientLayout /> : <AdminLayout />
}
