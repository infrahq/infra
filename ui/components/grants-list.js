import { sortByResource } from '../lib/grants'

import RoleSelect from './role-select'

export default function GrantsList({ grants, onRemove, onChange }) {
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
            onRemove={() => onRemove(g.id)}
            onChange={privilege => {
              onChange(privilege, g)
            }}
          />
        </div>
      ))}
    </>
  )
}
