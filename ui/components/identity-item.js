import { useState } from 'react'

import DeleteModal from './delete-modal'

export default function IdentityItem({
  userOrGroup,
  showDeleteModal,
  deleteModalInfo,
  onClick,
  showRemove = true,
  actionText = 'Remove',
}) {
  const [open, setOpen] = useState(false)

  function handleRemove() {
    if (showDeleteModal) {
      setOpen(true)
    } else {
      onClick()
    }
  }

  return (
    <>
      <div
        key={userOrGroup.id}
        className='group flex items-center justify-between truncate text-2xs'
      >
        <div className='py-2'>{userOrGroup.name}</div>
        {showRemove && (
          <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
            <button
              onClick={() => handleRemove()}
              className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
            >
              {actionText}
            </button>
            <DeleteModal
              open={open}
              setOpen={setOpen}
              primaryButtonText={deleteModalInfo?.btnText}
              onSubmit={onClick}
              title={deleteModalInfo?.title}
              message={deleteModalInfo?.message}
            />
          </div>
        )}
      </div>
    </>
  )
}
