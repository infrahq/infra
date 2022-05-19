import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { PlusIcon } from '@heroicons/react/outline'

import { validateEmail } from '../lib/email'

import InputDropdown from './input'
import DeleteModal from './modals/delete'
import ErrorMessage from './error-message'

function Grant ({ id, subject, grants }) {
  if (!id || !subject) {
    return null
  }

  const { data: user } = useSWR(`/v1/identities/${subject.replace('i:', '')}`, { fallbackData: { name: '', kind: '' } })
  const { data: auth } = useSWR('/v1/identities/self')
  const { mutate } = useSWRConfig()
  const [open, setOpen] = useState(false)

  const isSelf = subject.replace('i:', '') === auth.id

  return (
    <div className='flex group'>
      <div className='flex flex-1 items-center space-x-4 py-1'>
        <div className='border border-violet-300/20 flex-none flex items-center justify-center w-8 h-8 rounded-lg'>
          <div className='border border-violet-300/40 flex-none text-gray-500 flex justify-center items-center text-sm w-6 h-6 rounded-[4px]'>
            {user?.name?.[0]}
          </div>
        </div>
        <div className='flex flex-col leading-tight'>
          <div className='text-xs leading-none'>{user.name}</div>
        </div>
      </div>

      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        {!isSelf && <button onClick={() => setOpen(true)} className='flex-none p-2 -mr-2 cursor-pointer text-xs text-gray-500 hover:text-violet-100'>Revoke</button>}
        <DeleteModal
          open={open}
          setOpen={setOpen}
          onCancel={() => setOpen(false)}
          onSubmit={() => {
            mutate('/v1/grants?resource=infra&privilege=admin', async grants => {
              await fetch(`/v1/grants/${id}`, { method: 'DELETE' })

              return grants?.filter(g => g?.id !== id)
            }, { optimisticData: grants.filter(g => g?.id !== id) })

            setOpen(false)
          }}
          title='Revoke Admin'
          message={(<>Are you sure you want to revoke admin access for <span className='font-bold text-white'>{user.name}</span>?<br /><br /> This action cannot be undone.</>)}
        />
      </div>
    </div>
  )
}

export default function () {
  const { data: grants } = useSWR(() => '/v1/grants?resource=infra&privilege=admin', { fallbackData: [] })
  const { mutate } = useSWRConfig()

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

  console.log(grants)

  return (
    <div className='sm:w-80 lg:w-[500px]'>
      <div className='text-xs leading-none uppercase text-gray-400 border-b border-gray-800 pb-6'>Admins</div>
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
          className='flex items-center cursor-pointer border border-violet-300 px-5 mt-4 text-xs sm:ml-4 sm:mt-0 rounded-md disabled:pointer-events-none disabled:opacity-30'
        >
          <PlusIcon className='w-3 h-3 mr-1.5' />
          <div className='text-'>
            Add
          </div>
        </button>
      </div>
      {error &&
        <div className='mb-10'>
          <ErrorMessage message={error} />
        </div>}
      <h4 className='text-gray-400 my-3 text-xs'>These users have full administration privileges</h4>
      {grants.map(g => (
        <Grant key={g.id} id={g.id} subject={g.subject} grants={grants} />
      ))}
    </div>
  )
}
