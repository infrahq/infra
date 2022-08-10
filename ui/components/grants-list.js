import { sortByResource } from '../lib/grants'

import RoleSelect from './role-select'

export default function GrantsList({ grants, onRemove, onChange }) {
  return (
    <table className='min-w-full divide-y divide-gray-300'>
      <tbody className='bg-white'>
        {grants?.sort(sortByResource)?.map(g => (
          <tr key={g.id} className='border-b border-gray-200'>
            <td className='whitespace-nowrap py-4 text-xs font-medium'>
              <div className='truncate font-medium text-gray-900'>
                {g.resource}
              </div>
            </td>
            <td className='py-4 px-3 text-right text-sm text-gray-500'>
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
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
