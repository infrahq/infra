import { sortByPrivilege, sortBySubject } from '../lib/grants'
import dayjs from 'dayjs'
import Avatar from 'boring-avatars'

import { getAvatarName, iconsColors } from '../lib/icons'

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
            const subject = users?.find(u => u.id === info.row.original.user)
              ?.name
              ? users?.find(u => u.id === info.row.original.user)?.name
              : groups?.find(g => g.id === info.row.original.group)?.name
            return (
              <div className='flex flex-row items-center py-1'>
                <div className='mr-3'>
                  <Avatar
                    size={25}
                    name={getAvatarName(subject)}
                    variant={
                      users?.find(u => u.id === info.row.original.user)
                        ? 'beam'
                        : 'pixel'
                    }
                    colors={iconsColors}
                  />
                </div>
                <div className='flex flex-col py-0.5'>
                  <div className='text-sm font-medium text-gray-700'>
                    {users?.find(u => u.id === info.row.original.user)?.name}
                    {groups?.find(g => g.id === info.row.original.group)?.name}
                  </div>
                  <div className='text-2xs text-gray-500'>
                    {users?.find(u => u.id === info.row.original.user) &&
                      'User'}
                    {groups?.find(g => g.id === info.row.original.group)
                      ?.name && 'Group'}
                  </div>
                </div>
              </div>
            )
          },
        },
        {
          accessorKey: 'created',
          header: (
            <span className='hidden truncate lg:table-cell'>Last edited</span>
          ),
          cell: info => (
            <div className='hidden truncate lg:table-cell'>
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
