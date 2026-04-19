import { useState, useEffect, useCallback } from 'react'

/**
 * Generic CRUD page.
 *
 * Props:
 *   title        - page heading
 *   columns      - [{ key, label, render? }]  columns for the table
 *   idKey        - name of the primary-key field (default "id")
 *   fetchAll     - () => Promise<Array>
 *   fetchOne     - (id) => Promise<Object>   (optional, falls back to list item)
 *   create       - (data) => Promise<Object>
 *   update       - (id, data) => Promise<Object>
 *   remove       - (id) => Promise<void>
 *   fields       - field definitions for the form
 *   defaultValues - default form values
 */
export default function EntityPage({
  title,
  columns,
  idKey = 'id',
  fetchAll,
  create,
  update,
  remove,
  fields,
  defaultValues = {},
}) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [modal, setModal] = useState(null) // null | { mode: 'create'|'edit', data }
  const [form, setForm] = useState(defaultValues)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchAll()
      setRows(data || [])
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [fetchAll])

  useEffect(() => { load() }, [load])

  const openCreate = () => {
    setForm(defaultValues)
    setFormError(null)
    setModal({ mode: 'create' })
  }

  const openEdit = (row) => {
    const normalized = { ...row }
    fields.forEach((f) => {
      if (f.type === 'datetime-local' && normalized[f.key]) {
        try {
          normalized[f.key] = new Date(normalized[f.key]).toISOString().slice(0, 16)
        } catch {}
      }
    })
    setForm(normalized)
    setFormError(null)
    setModal({ mode: 'edit', id: row[idKey] })
  }

  const closeModal = () => setModal(null)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setSaving(true)
    setFormError(null)
    try {
      if (modal.mode === 'create') {
        await create(form)
      } else {
        await update(modal.id, form)
      }
      closeModal()
      load()
    } catch (e) {
      setFormError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this record?')) return
    try {
      await remove(id)
      load()
    } catch (e) {
      alert(e.message)
    }
  }

  return (
    <div className="p-4">
      <div className="d-flex align-items-center justify-content-between mb-3">
        <h4 className="mb-0">{title}</h4>
        <button className="btn btn-primary btn-sm" onClick={openCreate}>
          <i className="bi bi-plus-lg me-1" /> Add
        </button>
      </div>

      {error && <div className="alert alert-danger">{error}</div>}

      {loading ? (
        <div className="text-center py-5">
          <div className="spinner-border text-primary" />
        </div>
      ) : (
        <div className="card shadow-sm">
          <div className="table-responsive">
            <table className="table table-hover mb-0">
              <thead className="table-light">
                <tr>
                  {columns.map((c) => <th key={c.key}>{c.label}</th>)}
                  <th style={{ width: 100 }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.length === 0 ? (
                  <tr><td colSpan={columns.length + 1} className="text-center text-muted py-4">No records</td></tr>
                ) : rows.map((row) => (
                  <tr key={row[idKey]}>
                    {columns.map((c) => (
                      <td key={c.key} className="align-middle">
                        {c.render ? c.render(row[c.key], row) : String(row[c.key] ?? '')}
                      </td>
                    ))}
                    <td className="align-middle">
                      <button className="btn btn-sm btn-outline-secondary me-1" onClick={() => openEdit(row)}>
                        <i className="bi bi-pencil" />
                      </button>
                      <button className="btn btn-sm btn-outline-danger" onClick={() => handleDelete(row[idKey])}>
                        <i className="bi bi-trash" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {modal && (
        <div className="modal show d-block" tabIndex="-1" style={{ background: 'rgba(0,0,0,0.4)' }}>
          <div className="modal-dialog modal-dialog-centered">
            <div className="modal-content">
              <form onSubmit={handleSubmit}>
                <div className="modal-header">
                  <h5 className="modal-title">{modal.mode === 'create' ? `New ${title}` : `Edit ${title}`}</h5>
                  <button type="button" className="btn-close" onClick={closeModal} />
                </div>
                <div className="modal-body">
                  {formError && <div className="alert alert-danger py-2">{formError}</div>}
                  {fields.map((f) => (
                    <div className="mb-3" key={f.key}>
                      <label className="form-label fw-semibold">{f.label}</label>
                      {f.type === 'textarea' ? (
                        <textarea
                          className="form-control"
                          rows={3}
                          value={form[f.key] ?? ''}
                          onChange={(e) => setForm({ ...form, [f.key]: e.target.value })}
                          required={f.required}
                        />
                      ) : f.type === 'select' ? (
                        <select
                          className="form-select"
                          value={form[f.key] ?? ''}
                          onChange={(e) => setForm({ ...form, [f.key]: f.numericValue ? Number(e.target.value) : e.target.value })}
                          required={f.required}
                        >
                          <option value="">Select...</option>
                          {f.options?.map((o) => (
                            <option key={o.value} value={o.value}>{o.label}</option>
                          ))}
                        </select>
                      ) : f.type === 'checkbox' ? (
                        <div className="form-check">
                          <input
                            type="checkbox"
                            className="form-check-input"
                            checked={!!form[f.key]}
                            onChange={(e) => setForm({ ...form, [f.key]: e.target.checked })}
                          />
                        </div>
                      ) : (
                        <input
                          type={f.type || 'text'}
                          className="form-control"
                          value={form[f.key] ?? ''}
                          onChange={(e) => setForm({ ...form, [f.key]: f.type === 'number' ? Number(e.target.value) : e.target.value })}
                          onFocus={(e) => e.target.select()}
                          required={f.required}
                          min={f.min}
                          max={f.max}
                          step={f.step}
                        />
                      )}
                    </div>
                  ))}
                </div>
                <div className="modal-footer">
                  <button type="button" className="btn btn-secondary" onClick={closeModal}>Cancel</button>
                  <button type="submit" className="btn btn-primary" disabled={saving}>
                    {saving ? 'Saving…' : 'Save'}
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
