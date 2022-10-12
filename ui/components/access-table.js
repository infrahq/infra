import {
  ChevronDownIcon,
  ChevronUpIcon,
  XIcon,
  CheckIcon,
} from '@heroicons/react/solid'
import { Listbox, Disclosure, Transition } from '@headlessui/react'
import { useState } from 'react'
import { usePopper } from 'react-popper'
import * as ReactDOM from 'react-dom'

import { sortByRole, sortBySubject, descriptions } from '../lib/grants'

const OPTION_REMOVE = 'remove'

function NamespacesRolesComponent({ children }) {
  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button className='w-full'>
            <span className='flex items-center text-xs font-medium text-gray-500 '>
              <ChevronUpIcon
                className={`${
                  open ? 'rotate-180 transform' : ''
                } h-4 w-4 text-gray-500 duration-300 ease-in`}
              />
              Namespaces
            </span>
          </Disclosure.Button>
          <Transition
            show={open}
            enter='ease-out duration-1000'
            enterFrom='opacity-0'
            enterTo='opacity-100'
            leave='ease-in duration-300'
            leaveFrom='opacity-100'
            leaveTo='opacity-0'
          >
            <Disclosure.Panel static>{children}</Disclosure.Panel>
          </Transition>
        </>
      )}
    </Disclosure>
  )
}

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
        if (v.includes(OPTION_REMOVE)) {
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
          className='relative w-48 cursor-default rounded-md border border-gray-300 bg-white py-2 pl-3 pr-8 text-left text-xs shadow-sm hover:cursor-pointer hover:bg-gray-100 focus:outline-none'
        >
          <div className='flex space-x-1 truncate'>
            <span className='pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2 text-gray-700'>
              <ChevronDownIcon
                className='h-4 w-4 stroke-1 text-gray-700'
                aria-hidden='true'
              />
            </span>
            <span>{privileges[0]}</span>
            {privileges.length - 1 > 0 && (
              <span className='font-medium'> + {privileges.length - 1}</span>
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

function RoleList({ resource, privileges, roles, onUpdate }) {
  return (
    <div className='item-center flex justify-between space-x-2'>
      {resource && (
        <div className='py-2 text-xs font-medium text-gray-900'>
          {resource.split('.').pop()}
        </div>
      )}
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
}

function NamespacesRoleList({ reousrcesMap, roles, onUpdate }) {
  const namespacesRoleListComponent = []

  reousrcesMap.forEach((privileges, resource) =>
    namespacesRoleListComponent.push(
      <RoleList
        key={resource}
        resource={resource}
        privileges={sortByRole(privileges)}
        roles={roles}
        onUpdate={(v, r) => onUpdate(v, r)}
      />
    )
  )
  return namespacesRoleListComponent
}

function GrantCell({ grant, destination }) {
  const destinationPrivileges = grant.resourcePrivilegeMap.get(destination.name)

  const namespacesPrivilegeMap = new Map(
    Array.from(grant.resourcePrivilegeMap).filter(([key]) => {
      if (key.includes('.')) {
        return true
      }

      return false
    })
  )

  console.log('destinationPrivileges:', destinationPrivileges)
  console.log('namespace map:', namespacesPrivilegeMap)
  return (
    <div className='py-1'>
      {/* Destination Resource */}
      {destinationPrivileges?.length > 0 && (
        <div className='flex justify-between space-x-2 py-2'>
          <div className='py-2 text-xs font-medium text-gray-900'>
            cluster-wide access
          </div>
          <RoleList
            privileges={sortByRole(destinationPrivileges)}
            roles={destination.roles}
            onUpdate={v => {
              console.log(v)
            }}
          />
        </div>
      )}
      {/* Namespaces List */}
      {namespacesPrivilegeMap.size > 0 && (
        <div className='py-2'>
          <NamespacesRolesComponent>
            <div className='space-y-1'>
              <NamespacesRoleList
                reousrcesMap={namespacesPrivilegeMap}
                roles={destination.roles.filter(r => r != 'cluster-admin')}
                onUpdate={(v, r) => {
                  console.log(v)
                  console.log(r)
                }}
              />
            </div>
          </NamespacesRolesComponent>
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
    <div className='overflow-x-auto rounded-lg border border-gray-200/75'>
      <table className='w-full text-sm text-gray-600'>
        <thead className='border-b border-gray-200/75 bg-zinc-50/50 text-xs text-gray-500'>
          <tr>
            <th
              scope='col'
              className='py-2 px-5 text-left font-medium  sm:pl-6'
            >
              User or group
            </th>
            <th scope='col' className='py-2 px-5 text-left font-medium '>
              Roles
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
              <td className='w-[40%] whitespace-nowrap px-3 py-4 text-sm text-gray-500'>
                <GrantCell grant={grant} destination={destination} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
