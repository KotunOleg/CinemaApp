const BASE = ''

function getToken() {
  return localStorage.getItem('token') || ''
}

async function request(method, path, body, isForm = false) {
  const headers = {}
  if (!isForm) headers['Content-Type'] = 'application/json'
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const opts = { method, headers }
  if (body !== undefined) opts.body = isForm ? body : JSON.stringify(body)

  const res = await fetch(BASE + path, opts)
  if (res.status === 204) return null
  const text = await res.text()
  if (!res.ok) throw new Error(text || res.statusText)
  return text ? JSON.parse(text) : null
}

export async function downloadFile(path, filename) {
  const token = getToken()
  const res = await fetch(BASE + path, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) throw new Error(await res.text() || res.statusText)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  patch: (path, body) => request('PATCH', path, body),
  delete: (path) => request('DELETE', path),
  postForm: (path, formData) => request('POST', path, formData, true),
}
