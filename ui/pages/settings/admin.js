import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { useTable } from 'react-table'

import InputDropdown from '../../components/inputDropdown'
import Table from '../../components/table'
import DeleteModal from '../../components/modals/delete'
import { validateEmail } from '../../lib/email'

const columns = [{
  id: 'name',
  accessor: a => a,
  Cell: ({ value: admin }) => (
    <AdminName id={admin.subject} />
  )
}, {
  id: 'delete',
  accessor: a => a,
  Cell: ({ value: admin }) => {
    const { data: user } = useSWR(`/v1/identities/${admin.subject.replace('i:', '')}`, { fallbackData: { name: '', kind: '' } })
    const [open, setOpen] = useState(false)
    const { mutate } = useSWRConfig()

    return (
      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        <button onClick={() => setOpen(true)} className='p-2 -mr-2 cursor-pointer'>
          <p className='text-gray-500'>revoke admin</p>
        </button>
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onSubmit={() => {
            fetch(`/v1/grants/${admin.id}`, { method: 'DELETE' })
              .then(() => {
                setOpen(false)
              })
              .finally(() => {
                mutate('/v1/grants?resource=infra')
              })
              .catch((error) => {
                console.log(error)
              })
          }}
          title='Delete Admin'
          message={(<>Are you sure you want to delete <span className='font-bold text-white'>{user.name}</span>? This action cannot be undone.</>)}
        />
      </div>
    )
  }
}]

const AdminName = ({ id }) => {
  const { data: user } = useSWR(`/v1/identities/${id.replace('i:', '')}`, { fallbackData: { name: '', kind: '' } })
  console.log(user)
  return (
    <div className='flex items-center'>
      <div className='w-10 h-10 mr-4 bg-purple-100/10 font-bold rounded-lg flex items-center justify-center'>
        {user.name && user.name[0].toUpperCase()}
      </div>
      <div className='flex flex-col leading-tight'>
        <div className='font-medium'>{user.name}</div>
        <div className='text-gray-400 text-xs'>{user.kind}</div>
      </div>
    </div>
  )
}

export default function () {
  const { mutate } = useSWRConfig()
  const { data: adminList } = useSWR(() => '/v1/grants?resource=infra', { fallbackData: [] })
  const table = useTable({ columns, data: adminList || [] })

  const [adminEmail, setAdminEmail] = useState('')
  const [error, setError] = useState('')

  const grantAdminAccess = (id) => {
    fetch('/v1/grants', {
      method: 'POST',
      body: JSON.stringify({ subject: 'i:' + id, resource: 'infra', privilege: 'admin' })
    })
      .then(() => {
        mutate('/v1/grants?resource=infra')
        setAdminEmail('')
      }).catch((error) => {
        console.log(error)
      })
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
              body: JSON.stringify({ name: adminEmail, kind: 'user' })
            })
              .then((response) => response.json())
              .then((user) => {
                grantAdminAccess(user.id)
              })
              .catch((error) => {
                console.log(error)
              })
          } else {
            grantAdminAccess(data[0].id)
          }
        })
    } else {
      setError('Invalid email')
    }
  }

  return (
    <>
      <h3 className='text-lg font-bold mb-4'>Admins</h3>
      <h4 className='text-gray-300 mb-4 text-sm w-3/4'>Infra admins have full access to the Infra API, including creating additional grants, managing identity providers, managing destinations, and managing other users.</h4>
      <div className={`flex gap-1 ${error ? 'mt-10 mb-2' : 'my-10'} my-10 w-3/4`}>
        <div className='flex-1 w-full'>
          <InputDropdown
            type='email'
            value={adminEmail}
            placeholder='email'
            hasDropdownSelection={false}
            handleInputChange={e => handleInputChang(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
          />
        </div>
        <button
          onSubmit={() => handleAddAdmin()}
          disabled={adminEmail.length === 0}
          type='submit'
          className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mx-auto rounded-full'
        >
          <div className='bg-black flex items-center text-sm px-14 py-3 rounded-full'>
            Add
          </div>
        </button>
      </div>
      {error && <p className='text-sm text-pink-500'>{error}</p>}
      <h4 className='text-gray-400 my-3 text-sm'>These  users have full administration privileges</h4>
      {adminList && adminList.length > 0 &&
        <div className='w-3/4'>
          <Table {...table} showHeader={false} />
        </div>}
    </>
  )
}
