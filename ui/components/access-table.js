import { ChevronDownIcon, XIcon, CheckIcon } from '@heroicons/react/solid'
import { Listbox } from '@headlessui/react'
import { useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { sortByRole, sortBySubject, descriptions } from '../lib/grants'

import Table from './table'

const OPTION_REMOVE = 'remove'

function EditRoleMenu({
  roles,
  selectedRoles,
  onChange,
  onRemove,
  resource,
  privileges,
}) {
  roles = roles || []
  roles = sortByRole(roles)

  const [referenceElement, setReferenceElement] = useState(null)
  const [popperElement, setPopperElement] = useState(null)
  let { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: 'bottom-end',
    modifiers: [
      {
        name: 'flip',
        enabled: false,
      },
    ],
  })

  return (
    <Listbox
      value={selectedRoles}
      onChange={v => {
        console.log(v)
        if (v === OPTION_REMOVE) {
          onRemove()
          return
        }

        const newSelectedRole = v.filter(x => !selectedRoles.includes(x))

        onChange(newSelectedRole)
      }}
      multiple
    >
      <div className='relative'>
        <Listbox.Button
          ref={setReferenceElement}
          className='relative w-[15rem] cursor-default rounded-md border border-gray-300 bg-white py-2 pr-8 text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
        >
          <div className='flex space-x-1 truncate'>
            <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
              <ChevronDownIcon
                className='h-4 w-4 stroke-1 text-gray-700'
                aria-hidden='true'
              />
            </span>
            <span
              className={`inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-px text-2xs font-medium text-yellow-800`}
            >
              {privileges[0]}
            </span>
            {privileges[1] && (
              <span
                className={`inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-px text-2xs font-medium text-yellow-800`}
              >
                {privileges[1]}
              </span>
            )}
            {privileges.length - 2 > 0 && (
              <span className='text-gray-700'> + {privileges.length - 2}</span>
            )}
          </div>
        </Listbox.Button>
        {ReactDOM.createPortal(
          <Listbox.Options
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className={`sm:w-54 absolute z-[8] mt-2 max-h-64 w-48 overflow-auto rounded-md border border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none`}
          >
            {roles?.map(r => (
              <Listbox.Option
                key={r}
                className={({ active }) =>
                  `${
                    active ? 'bg-gray-100' : ''
                  } select-none py-2 px-3 hover:cursor-pointer`
                }
                value={r}
              >
                {({ selected }) => (
                  <div className='flex flex-row'>
                    <div className='flex flex-1 flex-col'>
                      <div className='flex justify-between py-0.5 font-medium'>
                        {r}
                        {selected && (
                          <CheckIcon
                            className='h-3 w-3 stroke-1 text-gray-600'
                            aria-hidden='true'
                          />
                        )}
                      </div>
                      <div className='text-3xs text-gray-600'>
                        {descriptions[r]}
                      </div>
                    </div>
                  </div>
                )}
              </Listbox.Option>
            ))}
            <Listbox.Option
              className={({ active }) =>
                `${
                  active ? 'bg-gray-50' : 'bg-white'
                } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-red-500`
              }
              value={OPTION_REMOVE}
            >
              <div className='flex flex-row items-center py-0.5'>
                <XIcon className='mr-1 mt-px h-3.5 w-3.5' /> Remove
              </div>
            </Listbox.Option>
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}

function RoleList({ resourcePrivilegeMap, roles, onUpdate }) {
  const roleListComponent = []
  resourcePrivilegeMap.forEach((privileges, resource) =>
    roleListComponent.push(
      <div
        className='item-center flex flex-col justify-between py-2'
        key={resource}
      >
        <div className='text-xs font-medium text-gray-900'>{resource}</div>
        <EditRoleMenu
          roles={roles}
          selectedRoles={privileges}
          onChange={v => {
            console.log(resource)
            onUpdate(v, resource)
          }}
          onRemove={() => {}}
          resource={resource}
          privileges={privileges}
        />
      </div>
    )
  )
  return roleListComponent
}

export default function AccessTable({
  grants,
  users,
  groups,
  destination,
  onUpdate,
  onRemove,
  onChange,
}) {
  const grantsSubject = [...new Set(grants?.map(g => g.user || g.group))]
  let grantsList = []
  grantsSubject.forEach(subject => {
    let type = 'user'
    const grantArray = grants.filter(g => {
      if (g.group === subject) {
        type = 'group'
      }
      return g.user === subject || g.group === subject
    })

    const resourcePrivilegeMap = new Map()
    grantArray.forEach(g => {
      if (resourcePrivilegeMap.has(g.resource)) {
        resourcePrivilegeMap.set(g.resource, [
          ...resourcePrivilegeMap.get(g.resource),
          g.privilege,
        ])
      } else {
        resourcePrivilegeMap.set(g.resource, [g.privilege])
      }
    })

    if (grantArray.length === 1) {
      grantArray[0].resourcePrivilegeMap = resourcePrivilegeMap
      grantsList = [...grantsList, ...grantArray]
    } else {
      grantsList.push({
        [type]: subject,
        id: grantArray.map(g => g.id),
        resourcePrivilegeMap,
      })
    }
  })

  console.log(grants)
  console.log(grantsList)

  return (
    <Table
      data={grantsList?.sort(sortBySubject)}
      columns={[
        {
          id: 'subject',
          header: 'User or group',
          cell: function Cell(info) {
            return (
              <div className='flex w-[60%] flex-col truncate'>
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
          id: 'role',
          cell: function Cell(info) {
            return (
              <RoleList
                resourcePrivilegeMap={info.row.original.resourcePrivilegeMap}
                roles={destination.roles}
                onUpdate={(newPrivilege, resource) =>
                  onUpdate(
                    newPrivilege[0],
                    info.row.original.user,
                    info.row.original.group,
                    resource
                  )
                }
              />
            )
          },
        },
      ]}
    />
  )
}
