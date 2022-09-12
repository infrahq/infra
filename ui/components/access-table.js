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
  inherited,
}) {
  return (
    <Table
      data={grants?.sort(sortByPrivilege)?.sort(sortBySubject)}
      columns={[
        {
          id: 'subject',
          header: 'User or Group',
          cell: function Cell(info) {
            return (
              <>
                {users?.find(u => u.id === info.row.original.user)?.name}
                {groups?.find(g => g.id === info.row.original.group)?.name}
              </>
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
    // <table
    //   className={`${
    //     inherited && inherited.length > 0 && grants.length === 0 ? 'mt-2' : ''
    //   } min-w-full divide-y divide-gray-300`}
    // >
    //   <tbody className='bg-white'>
    //     {grants
    //       ?.sort(sortByPrivilege)
    //       ?.sort(sortBySubject)
    //       ?.map(group => (
    //         <tr key={group.id} className='border-b border-gray-200'>
    //           <td className='whitespace-nowrap py-4'>
    //             <div className='truncate text-sm font-medium text-gray-900'>
    //               {users?.find(u => u.id === group.user)?.name}
    //               {groups?.find(g => g.id === group.group)?.name}
    //             </div>
    //           </td>
    //           <td className='py-4 px-3 text-right'>
    //             <RoleSelect
    //               role={group.privilege}
    //               roles={destination.roles}
    //               remove
    //               onRemove={async () => onRemove(group.id)}
    //               onChange={async privilege => onChange(privilege, group)}
    //               direction='left'
    //             />
    //           </td>
    //         </tr>
    //       ))}
    //     {inherited && inherited.length > 0 && (
    //       <>
    //         <tr>
    //           <th
    //             colSpan={5}
    //             scope='colgroup'
    //             title='This access is inherited by a group and cannot be edited here'
    //             className='bg-gray-100 p-2 text-left text-sm font-semibold text-gray-900'
    //           >
    //             Inherited
    //           </th>
    //         </tr>
    //         {inherited
    //           ?.sort(sortByPrivilege)
    //           ?.sort(sortBySubject)
    //           ?.map(item => (
    //             <tr key={item.id} className='border-b border-gray-200'>
    //               <td className='whitespace-nowrap py-4 text-sm font-medium'>
    //                 <div className='truncate font-medium text-gray-900'>
    //                   {users?.find(u => u.id === item.user)?.name}
    //                   {groups?.find(group => group.id === item.group)?.name}
    //                 </div>
    //               </td>
    //               <td className='py-4 px-3 text-right text-sm text-gray-700'>
    //                 {item.privilege}
    //               </td>
    //             </tr>
    //           ))}
    //       </>
    //     )}
    //   </tbody>
    // </table>
  )
}
