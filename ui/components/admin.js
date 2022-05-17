import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { useTable } from 'react-table'
import { PlusIcon } from '@heroicons/react/outline'

import { validateEmail } from '../lib/email'

import InputDropdown from './input'
import Table from './table'
import DeleteModal from './modals/delete'
import ErrorMessage from './error-message'

const columns = [{
  id: 'name',
  accessor: a => a,
  Cell: ({ value: admin }) => (
    <AdminName id={admin.subject} />
  )
}, {
  id: 'delete',
  accessor: a => a,
  Cell: ({ value: admin, rows }) => {
    const { data: user } = useSWR(`/v1/identities/${admin.subject.replace('i:', '')}`, { fallbackData: { name: '', kind: '' } })
    const { data: auth } = useSWR('/v1/identities/self')
    const { mutate } = useSWRConfig()

    const [open, setOpen] = useState(false)

    const isSelf = admin.subject.replace('i:', '') === auth.id

    return (
      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        {!isSelf && <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer text-gray-500 hover:text-white'>Revoke</button>}
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onCancel={() => setOpen(false)}
          onSubmit={() => {
            mutate('/v1/grants?resource=infra&privilege=admin', async admins => {
              await fetch(`/v1/grants/${admin.id}`, { method: 'DELETE' })

              return admins?.filter(a => a?.id !== admin.id)
            }, { optimisticData: rows.map(r => r.original).filter(a => a?.id !== admin.id) })

            setOpen(false)
          }}
          title='Delete Admin'
          message={(<>Are you sure you want to delete <span className='font-bold text-white'>{user.name}</span>? This action cannot be undone.</>)}
        />
      </div>
    )
  }
}]

const AdminName = ({ id }) => {
  if (!id) {
    return null
  }

  const { data: user } = useSWR(`/v1/identities/${id.replace('i:', '')}`, { fallbackData: { name: '', kind: '' } })

  return (
    <div className='flex items-center space-x-4'>
      <div className='bg-gradient-to-tr from-indigo-300/20 to-pink-100/20 rounded-lg p-px'>
        <div className='bg-black flex-none flex items-center justify-center w-8 h-8 rounded-lg'>
          <div className='bg-gradient-to-tr from-indigo-300/40 to-pink-100/40 rounded-[4px] p-px'>
            <div className='bg-black flex-none text-gray-500 flex justify-center items-center w-6 h-6 font-bold rounded-[4px]'>
              {user?.name?.[0]}
            </div>
          </div>
        </div>
      </div>
      <div className='flex flex-col leading-tight'>
        <div className='text-subtitle'>{user.name}</div>
      </div>
    </div>
  )
}

export default function () {
  const { data: adminList } = useSWR(() => '/v1/grants?resource=infra&privilege=admin', { fallbackData: [] })
  const { mutate } = useSWRConfig()

  const table = useTable({ columns, data: adminList || [] })

  const [adminEmail, setAdminEmail] = useState('')
  const [error, setError] = useState('')

  const grantAdminAccess = (id) => {
    fetch('/v1/grants', {
      method: 'POST',
      body: JSON.stringify({ subject: 'i:' + id, resource: 'infra', privilege: 'admin' })
    })
      .then(() => {
        mutate('/v1/grants?resource=infra&privilege=admin')
        setAdminEmail('')
      }).catch((e) => setError(e.message || 'something went wrong, please try again later.'))
  }

  const handleInputChang = (value) => {
    setAdminEmail(value)
    setError('')
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && adminEmail.length > 0) {
      handleAddAdmin()
    }
  }

  const handleAddAdmin = () => {
    if (validateEmail(adminEmail)) {
      setError('')

      fetch(`/v1/identities?name=${adminEmail}`)
        .then((response) => response.json())
        .then((data) => {
          if (data.length === 0) {
            fetch('/v1/identities', {
              method: 'POST',
              body: JSON.stringify({ name: adminEmail })
            })
              .then((response) => response.json())
              .then((user) => grantAdminAccess(user.id))
              .catch((error) => console.error(error))
          } else {
            grantAdminAccess(data[0].id)
          }
        })
    } else {
      setError('Invalid email')
    }
  }

  return (
    <div className='sm:w-80 lg:w-[500px]'>
      <div className='text-subtitle uppercase text-gray-400 border-b border-gray-800 pb-6'>Admins</div>
      <div className={`flex flex-col sm:flex-row ${error ? 'mt-6 mb-2' : 'mt-6 mb-14'}`}>
        <div className='sm:flex-1'>
          <InputDropdown
            type='email'
            value={adminEmail}
            placeholder='Email address'
            hasDropdownSelection={false}
            handleInputChange={e => handleInputChang(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            error={error}
          />
        </div>
        <button
          onClick={() => handleAddAdmin()}
          disabled={adminEmail.length === 0}
          type='button'
          className='bg-gradient-to-tr disabled:opacity-30 disabled:transform-none disabled:transition-none cursor-pointer disabled:cursor-default from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mt-4 mr-auto sm:ml-4 sm:mt-0 rounded-md'
        >
          <div className='bg-black flex items-center text-xs rounded-md hover:text-pink-50 px-6 py-3'>
            <PlusIcon className='w-3 h-3 mr-1.5' />
            <div className='text-pink-100'>
              Add
            </div>
          </div>
        </button>
      </div>
      {
        error &&
          <div className='mb-10'>
            <ErrorMessage message={error} />
          </div>
      }

      <h4 className='text-gray-400 my-3 text-paragraph'>These users have full administration privileges</h4>
      {adminList?.length > 0 &&
        <Table {...table} showHeader={false} />}
    </div>
  )
}
