import { useEffect, useState } from 'react'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import { Bar, Pie } from 'react-chartjs-2'
import { api } from '../api/client'

ChartJS.register(CategoryScale, LinearScale, BarElement, ArcElement, Title, Tooltip, Legend)

const PALETTE = [
  '#4e79a7', '#f28e2b', '#e15759', '#76b7b2', '#59a14f',
  '#edc948', '#b07aa1', '#ff9da7', '#9c755f', '#bab0ac',
]

export default function StatisticsPage() {
  const [byGenre, setByGenre] = useState([])
  const [avgRating, setAvgRating] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    Promise.all([
      api.get('/api/stats/movies-by-genre'),
      api.get('/api/stats/avg-rating'),
    ])
      .then(([genre, rating]) => {
        setByGenre(genre || [])
        setAvgRating(rating || [])
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="d-flex justify-content-center align-items-center p-5">
        <div className="spinner-border text-primary" />
      </div>
    )
  }

  if (error) {
    return <div className="alert alert-danger m-4">{error}</div>
  }

  const genreData = {
    labels: byGenre.map((r) => r.genre),
    datasets: [
      {
        label: 'Number of Movies',
        data: byGenre.map((r) => r.count),
        backgroundColor: byGenre.map((_, i) => PALETTE[i % PALETTE.length]),
        borderRadius: 4,
      },
    ],
  }

  const ratingData = {
    labels: avgRating.map((r) => r.title),
    datasets: [
      {
        label: 'Average Rating',
        data: avgRating.map((r) => r.avg_rating),
        backgroundColor: avgRating.map((_, i) => PALETTE[i % PALETTE.length]),
        borderRadius: 4,
      },
    ],
  }

  const pieData = {
    labels: byGenre.map((r) => r.genre),
    datasets: [
      {
        data: byGenre.map((r) => r.count),
        backgroundColor: byGenre.map((_, i) => PALETTE[i % PALETTE.length]),
      },
    ],
  }

  return (
    <div className="p-4">
      <h4 className="fw-bold mb-4">
        <i className="bi bi-bar-chart-fill me-2 text-primary" />
        Statistics
      </h4>

      <div className="row g-4">
        <div className="col-12 col-xl-7">
          <div className="card shadow-sm h-100">
            <div className="card-header bg-white fw-semibold">Movies by Genre</div>
            <div className="card-body">
              {byGenre.length === 0 ? (
                <p className="text-muted">No genre data available.</p>
              ) : (
                <Bar
                  data={genreData}
                  options={{
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } },
                  }}
                />
              )}
            </div>
          </div>
        </div>

        <div className="col-12 col-xl-5">
          <div className="card shadow-sm h-100">
            <div className="card-header bg-white fw-semibold">Genre Distribution</div>
            <div className="card-body d-flex justify-content-center align-items-center">
              {byGenre.length === 0 ? (
                <p className="text-muted">No genre data available.</p>
              ) : (
                <div style={{ maxWidth: 320, width: '100%' }}>
                  <Pie data={pieData} options={{ responsive: true }} />
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="col-12">
          <div className="card shadow-sm">
            <div className="card-header bg-white fw-semibold">Top Movies by Average Rating</div>
            <div className="card-body">
              {avgRating.length === 0 ? (
                <p className="text-muted">No rating data available. Add some reviews first.</p>
              ) : (
                <Bar
                  data={ratingData}
                  options={{
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: {
                      y: { beginAtZero: true, max: 10, ticks: { stepSize: 1 } },
                      x: { ticks: { maxRotation: 30 } },
                    },
                  }}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
