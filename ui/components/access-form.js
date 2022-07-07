import { sortByResource } from '../lib/grants'

import RoleSelect from './role-select'

export default function AccessForm({ grants, mutate }) {
  return (
    <>
      {grants?.sort(sortByResource)?.map(g => (
        <div key={g.id} className='flex items-center justify-between text-2xs'>
          <div>{g.resource}</div>
          <RoleSelect
            role={g.privilege}
            resource={g.resource}
            remove
            direction='left'
            onRemove={async () => {
              await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })
              mutate({ items: grants.filter(x => x.id !== g.id) })
            }}
            onChange={async privilege => {
              const res = await fetch('/api/grants', {
                method: 'POST',
                body: JSON.stringify({
                  ...g,
                  privilege,
                }),
              })

              // delete old grant
              await fetch(`/api/grants/${g.id}`, { method: 'DELETE' })

              mutate({
                items: [...grants.filter(f => f.id !== g.id), await res.json()],
              })
            }}
          />
        </div>
      ))}
    </>
  )
}
