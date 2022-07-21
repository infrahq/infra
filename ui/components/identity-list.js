import { useState } from 'react'

import DeleteModal from './delete-modal'

export default function IdentityList({
  list,
  authId,
  deleteModal,
  onClick,
  showRemove = true,
  actionText = 'Remove',
}) {
  const [open, setOpen] = useState(false)

  function handleRemove(identityId) {
    if (identityId === authId) {
      setOpen(true)
    } else {
      onClick(identityId)
    }
  }
  return (
    <>
      {list
        ?.sort((a, b) => b.created?.localeCompare(a.created))
        ?.map(i => (
          <div
            key={i.id}
            className='group flex items-center justify-between truncate text-2xs'
          >
            <div className='py-2'>{i.name}</div>
            {showRemove && (
              <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
                <button
                  onClick={() => handleRemove(i.id)}
                  className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
                >
                  {actionText}
                </button>
                <DeleteModal
                  open={open}
                  setOpen={setOpen}
                  primaryButtonText={deleteModal?.btnText}
                  onSubmit={onClick}
                  title={deleteModal?.title}
                  message={deleteModal?.message}
                />
              </div>
            )}
          </div>
        ))}
    </>
  )
}
