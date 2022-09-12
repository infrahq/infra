import { sortByPrivilege, sortBySubject } from '../lib/grants'
import dayjs from 'dayjs'

import Table from './table'
import RoleSelect from './role-select'

export default function AccessTable({
  grants,
  users,
  groups,
  destination,
  onRemove,
  onChange,
}) {
  return (
    <Table
      data={grants?.sort(sortByPrivilege)?.sort(sortBySubject)}
      columns={[
        {
          id: 'subject',
          header: 'User or group',
          cell: function Cell(info) {
            return (
              <div className='flex flex-col'>
                <div className='text-sm font-medium text-gray-700'>
                  {users?.find(u => u.id === info.row.original.user)?.name}
                  {groups?.find(g => g.id === info.row.original.group)?.name}
                </div>
                <div className='text-2xs text-gray-500'>
                  {users?.find(u => u.id === info.row.original.user) && 'User'}
                  {groups?.find(g => g.id === info.row.original.group)?.name &&
                    'Group'}
                </div>
              </div>
            )
          },
        },
        {
          accessorKey: 'created',
          header: 'Last edited',
          cell: info => (
            <div className='truncate'>
              {info.getValue() ? dayjs(info.getValue()).fromNow() : '-'}
            </div>
          ),
        },
        {
          id: 'role',
          cell: info => (
            <div className='overflow-visible text-right'>
              <RoleSelect
                role={info.row.original.privilege}
                roles={destination.roles}
                remove
                onRemove={async () => onRemove(info.row.original.id)}
                onChange={async privilege =>
                  onChange(privilege, info.row.original)
                }
                direction='left'
              />
            </div>
          ),
        },
      ]}
    />
  )
}
