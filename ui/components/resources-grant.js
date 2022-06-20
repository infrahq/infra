import useSWR, { mutate } from 'swr'

export default function ({ user }) {
  const { data: { items } = {} } = useSWR(`/api/grants?user=${user}`)

  const options = ['view', 'edit', 'admin', 'remove']

  function grant (privilege, resource) {
    mutate(`/api/grants?user=${user}`, async ({ items: grants } = { items: [] }) => {
      const res = await fetch('/api/grants', {
        method: 'POST',
        body: JSON.stringify({ user, resource, privilege })
      })

      const data = await res.json()

      // replace any existing grants
      const existing = grants.filter(g => g.resource === resource)
      for (const e of existing) {
        await fetch(`/api/grants/${e.id}`, { method: 'DELETE' })
      }

      return { items: [...grants.filter(g => g.resource !== resource), data] }
    })
  }

  async function remove (id) {
    mutate(`/api/grants?user=${user}`, async ({ items: grants } = { items: [] }) => {
      await fetch(`/api/grants/${id}`, { method: 'DELETE' })
      return { items: grants?.filter(item => item?.id !== id) }
    }, { optimisticData: { items: items?.filter(item => item?.id !== id) } })
  }

  return (
    <>
      {items?.length > 0 &&
        <div className='py-2 max-h-40 overflow-y-auto'>
          {items?.filter(grant => grant.resource !== 'infra').sort((a, b) => b.resource?.localeCompare(a.resource)).map(item => (
            <div className='flex justify-between items-center' key={item.id}>
              <p className='text-2xs'>{item.resource}</p>
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-2xs text-gray-400 bg-transparent'
                  defaultValue={item.privilege}
                  onChange={({ target: { value: privilege } }) => {
                    if (privilege === 'remove') {
                      remove(item.id)
                      return
                    }

                    grant(privilege, item.resource)
                  }}
                >
                  {options.map((option) => (
                    <option key={option} value={option}>{option}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>}
      {items?.filter(grant => grant.resource !== 'infra').length === 0 &&
        <div className='text-2xs text-gray-400 italic w-2/3 py-4'>
          No access
        </div>}
    </>
  )
}
