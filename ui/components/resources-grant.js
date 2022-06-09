import { useState } from 'react'
import useSWR, { mutate } from 'swr'

export default function ({ id }) {
  const { data: { items: grants } = {} } = useSWR(`/api/grants?user=${id}`)

  const [infrastructure, setInfrastructure] = useState('')
  const [role, setRole] = useState('view')

  const options = ['view', 'edit', 'admin', 'remove']

  function grant (user, privilege = role, resource = infrastructure, exist = false, deleteGrantId) {
    mutate(`/api/grants?user=${id}`, async ({ items: grantsList } = { items: [] }) => {
      const res = await fetch('/api/grants', {
        method: 'POST',
        body: JSON.stringify({ user, resource, privilege })
      })

      const data = await res.json()

      setInfrastructure('')

      if (exist) {
        await fetch(`/api/grants/${deleteGrantId}`, { method: 'DELETE' })
        return { items: [...grantsList.filter(grant => grant?.user !== user), data] }
      }

      return { items: [...grantsList, data] }
    })
  }

  function handleUpdateGrant (privilege, resource, grantId, user) {
    setRole(privilege)
    if (privilege !== 'remove') {
      return grant(user, privilege, resource, true, grantId)
    }

    mutate(`/api/grants?user=${user}`, async ({ items: userGrantsList } = { items: [] }) => {
      await fetch(`/api/grants/${grantId}`, { method: 'DELETE' })
      return { items: userGrantsList?.filter(item => item?.id !== grantId) }
    }, { optimisticData: { items: grants?.filter(g => g?.id !== grantId) } })
  }

  return (
    <>
      {grants?.length > 0 &&
        <div className='py-2'>
          {grants?.filter(grant => grant.resource !== 'infra').sort((a, b) => b.id.localeCompare(a.id)).map(item => (
            <div className='flex justify-between items-center' key={item.id}>
              <p className='text-2xs'>{item.resource}</p>
              <div>
                <select
                  id='role'
                  name='role'
                  className='w-full pl-3 pr-1 py-2 border-gray-300 focus:outline-none text-2xs text-gray-400 bg-transparent'
                  defaultValue={item.privilege}
                  onChange={e => handleUpdateGrant(e.target.value, item.resource, item.id, item.user)}
                >
                  {options.map((option) => (
                    <option key={option} value={option}>{option}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>}
      {grants?.filter(grant => grant.resource !== 'infra').length === 0 &&
        <div className='text-2xs text-gray-400 italic w-2/3 py-2'>
          No access
        </div>}
    </>
  )
}
