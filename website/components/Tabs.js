import { useState, createContext } from 'react'

export const TabContext = createContext()

export default function Tabs({ labels, children }) {
  const [currentTab, setCurrentTab] = useState(labels[0])

  return (
    <TabContext.Provider value={currentTab}>
      <div className='flex flex-wrap'>
        {labels.map(label => (
          <button
            key={label}
            className={`${
              label === currentTab
                ? 'border-zinc-100 text-white'
                : 'border-transparent text-zinc-500'
            } border-b px-7 py-1.5 text-lg font-medium leading-loose first-of-type:ml-0`}
            onClick={() => setCurrentTab(label)}
          >
            {label}
          </button>
        ))}
      </div>
      {children}
    </TabContext.Provider>
  )
}
