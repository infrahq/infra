export default function IdentityList({ list, actionText = 'Remove', onClick }) {
  return (
    <>
      {list
        ?.sort((a, b) => b.created?.localeCompare(a.created))
        ?.map(u => (
          <div
            key={u.id}
            className='group flex items-center justify-between truncate text-2xs'
          >
            <div className='py-2'>{u.name}</div>
            <div className='flex justify-end text-right opacity-0 group-hover:opacity-100'>
              <button
                onClick={() => onClick(u.id)}
                className='-mr-2 flex-none cursor-pointer px-2 py-1 text-2xs text-gray-500 hover:text-violet-100'
              >
                {actionText}
              </button>
            </div>
          </div>
        ))}
    </>
  )
}
