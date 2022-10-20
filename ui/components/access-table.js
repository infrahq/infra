import { ChevronDownIcon, XIcon, CheckIcon } from '@heroicons/react/solid'
import { Listbox } from '@headlessui/react'
import { useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { sortByRole, sortBySubject, descriptions } from '../lib/grants'

import DisclosureForm from './disclosure-form'

const OPTION_REMOVE = 'remove'

function EditRoleMenu({
  roles,
  selectedRoles,
  onChange,
  onRemove,
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
      {
        name: 'offset',
        options: {
          offset: [0, 5],
        },
      },
    ],
  })

  return (
    <Listbox
      value={selectedRoles}
      onChange={v => {
        if (v.includes(OPTION_REMOVE)) {
          onRemove()
          return
        }

        if (selectedRoles.length === 1 && v.length === 0) {
          return
        }

        onChange(v)
      }}
      multiple
    >
      <div className='relative'>
        <Listbox.Button
          ref={setReferenceElement}
          className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
        >
          <div className='flex space-x-1 truncate'>
            <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2'>
              <ChevronDownIcon
                className='h-4 w-4 stroke-1 text-gray-700'
                aria-hidden='true'
              />
            </span>
            <span className='text-gray-700'>{privileges?.[0]}</span>
            {privileges.length > 1 && (
              <span className='font-medium'> + {privileges.length - 1}</span>
            )}
          </div>
        </Listbox.Button>
        {ReactDOM.createPortal(
          <Listbox.Options
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
            className='absolute z-10 w-48 overflow-auto rounded-md border  border-gray-200 bg-white text-left text-xs text-gray-800 shadow-lg shadow-gray-300/20 focus:outline-none'
          >
            <div className='max-h-64 overflow-auto'>
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
            </div>
            <Listbox.Option
              className={({ active }) =>
                `${
                  active ? 'bg-gray-50' : 'bg-white'
                } group flex w-full items-center border-t border-gray-100 px-2 py-1.5 text-xs font-medium text-red-500 hover:cursor-pointer`
              }
              value={OPTION_REMOVE}
            >
              <div className='flex flex-row items-center py-0.5'>
                <XIcon className='mr-1 mt-px h-3.5 w-3.5' /> Remove access
              </div>
            </Listbox.Option>
          </Listbox.Options>,
          document.querySelector('body')
        )}
      </div>
    </Listbox>
  )
}

function RoleList({ resource, privileges, roles, onUpdate, onRemove }) {
  return (
    <div className='item-center flex justify-between'>
      {resource && (
        <div className='block w-1/2 truncate py-2 px-4 text-xs font-medium text-gray-900'>
          {resource.split('.').pop()}
        </div>
      )}
      <EditRoleMenu
        roles={roles}
        selectedRoles={privileges}
        onChange={v => {
          onUpdate(v)
        }}
        onRemove={() => {
          onRemove()
        }}
        resource={resource}
        privileges={privileges}
      />
    </div>
  )
}

function NamespacesRoleList({ reousrcesMap, roles, onUpdate, onRemove }) {
  const namespacesRoleListComponent = []

  reousrcesMap.forEach((privileges, resource) =>
    namespacesRoleListComponent.push(
      <RoleList
        key={resource}
        resource={resource}
        privileges={sortByRole(privileges)}
        roles={roles}
        onUpdate={v => onUpdate(v, resource, privileges)}
        onRemove={() => onRemove(resource)}
      />
    )
  )
  return namespacesRoleListComponent
}

function GrantCell({ grantsList, grant, destination, onRemove, onUpdate }) {
  const destinationPrivileges = grant.resourcePrivilegeMap.get(destination.name)

  const namespacesPrivilegeMap = new Map(
    Array.from(grant.resourcePrivilegeMap).filter(([key]) => {
      if (key.includes('.')) {
        return true
      }

      return false
    })
  )

  function handleRemove(resource) {
    const deleteGrantIdList = grantsList
      .filter(g => g.resource === resource)
      .filter(g => g.user === grant.user)
      .filter(g => g.group === grant.group)
      .map(g => g.id)

    onRemove(deleteGrantIdList)
  }

  function handleUpdate(newPrivilege, selectedPrivilege, resource) {
    // update to add roles
    if (newPrivilege.length > selectedPrivilege.length) {
      const newRole = newPrivilege.filter(x => !selectedPrivilege.includes(x))
      onUpdate(newRole, resource)
    } else {
      // update to delete roles
      const removeRoles = selectedPrivilege.filter(
        x => !newPrivilege.includes(x)
      )

      const deleteGrantIdList = grantsList
        .filter(g => g.resource === resource)
        .filter(g => g.user === grant.user)
        .filter(g => g.group === grant.group)
        .filter(g => removeRoles.includes(g.privilege))
        .map(g => g.id)
      onRemove(deleteGrantIdList)
    }
  }

  return (
    <div className='py-1'>
      {/* Destination Resource */}
      {destinationPrivileges?.length > 0 && (
        <div className='flex justify-end space-x-2 py-2'>
          <RoleList
            privileges={sortByRole(destinationPrivileges)}
            roles={destination.roles}
            onUpdate={v =>
              handleUpdate(v, destinationPrivileges, destination.name)
            }
            onRemove={() => handleRemove(destination.name)}
          />
        </div>
      )}
      {/* Namespaces List */}
      {namespacesPrivilegeMap.size > 0 && (
        <div className='py-2'>
          <DisclosureForm
            title={`Namespace access (${namespacesPrivilegeMap.size})`}
            defaultOpen={destinationPrivileges === undefined}
          >
            <div className='space-y-2 pt-2'>
              <NamespacesRoleList
                reousrcesMap={namespacesPrivilegeMap}
                roles={destination.roles.filter(r => r != 'cluster-admin')}
                onUpdate={(v, resource, namespacesPrivileges) =>
                  handleUpdate(v, namespacesPrivileges, resource)
                }
                onRemove={resource => handleRemove(resource)}
              />
            </div>
          </DisclosureForm>
        </div>
      )}
    </div>
  )
}

export default function AccessTable({
  grants,
  users,
  groups,
  destination,
  onUpdate,
  onRemove,
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

  return (
    <div className='overflow-x-auto rounded-lg border border-gray-200/75'>
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          <tr>
            <th
              scope='col'
              className='py-2 px-5 text-left font-medium  sm:pl-6'
            >
              Group or user
            </th>
          </tr>
        </thead>
        <tbody className='divide-y divide-gray-200 bg-white'>
          {grantsList?.sort(sortBySubject).map(grant => (
            <tr key={grant.user || grant.group}>
              <td className='w-[60%] whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6'>
                <div className='flex w-[60%] flex-col truncate'>
                  <div className='text-sm font-medium text-gray-700'>
                    {users?.find(u => u.id === grant.user)?.name}
                    {groups?.find(g => g.id === grant.group)?.name}
                  </div>
                  <div className='text-2xs text-gray-500'>
                    {users?.find(u => u.id === grant.user) && 'User'}
                    {groups?.find(g => g.id === grant.group)?.name && 'Group'}
                  </div>
                </div>{' '}
              </td>
              <td className='w-[35%] whitespace-nowrap px-3 py-4 text-sm text-gray-500'>
                <GrantCell
                  grantsList={grants}
                  grant={grant}
                  destination={destination}
                  onRemove={grantsIdList => onRemove(grantsIdList)}
                  onUpdate={(newPrivilege, resource) =>
                    onUpdate(newPrivilege, grant.user, grant.group, resource)
                  }
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {grantsList && grantsList.length === 0 && (
        <div className='flex justify-center py-5 text-sm text-gray-500'>
          No data
        </div>
      )}
      {!grantsList && (
        <div className='flex min-h-[100px] items-center justify-center py-4 text-xs text-gray-400'>
          <svg
            xmlns='http://www.w3.org/2000/svg'
            viewBox='0 0 100 100'
            preserveAspectRatio='xMidYMid'
            className='h-10 w-10 animate-spin-fast stroke-current text-gray-400'
          >
            <circle
              cx='50'
              cy='50'
              fill='none'
              strokeWidth='1.5'
              r='24'
              strokeDasharray='113.09733552923255 39.69911184307752'
            ></circle>
          </svg>
        </div>
      )}
    </div>
  )
}
