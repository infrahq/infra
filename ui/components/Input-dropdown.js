export default function InputDropdown ({
  label,
  type,
  value,
  placeholder,
  error,
  hasDropdownSelection = true,
  optionType,
  options,
  handleInputChange,
  handleSelectOption,
  handleKeyDown,
}) {
  return (
    <div>
      {label &&
        <label htmlFor='price' className='block text-sm font-medium text-white'>
          {label}
        </label>}
      <div className='relative rounded shadow-sm'>
        <input
          type={type}
          value={value}
          className={`block w-full px-4 py-3 sm:text-sm border-2 bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600 ${error ? 'border-pink-500' : 'border-gray-800'}`}
          placeholder={placeholder}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
        />
        {hasDropdownSelection &&
          <div className='absolute inset-y-0 right-2 flex items-center'>
            <label htmlFor={optionType} className='sr-only'>
              {optionType}
            </label>
            <select
              id={optionType}
              name={optionType}
              onChange={handleSelectOption}
              className='h-full py-0 pl-2 border-transparent bg-transparent text-white text-sm focus:outline-none'
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>}
      </div>
    </div>

  /* <form onSubmit={onSubmit} className='flex gap-1 my-10 w-full'>
{label &&
  <label htmlFor='price' className='block text-sm font-medium text-white'>
    {label}
  </label>}
    <div className='relative rounded shadow-sm flex-1 w-full'>
  <input
    autoFocus
    required={required}
    type={type}
    value={value}
    className='block w-full px-4 py-3 sm:text-sm border-2 border-gray-800 bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600'
    placeholder={placeholder}
    onChange={handleInputChange}
    onKeyDown={handleKeyDown}
  />
  {hasDropdownSelection &&
    <div className='absolute inset-y-0 right-2 flex items-center'>
      <label htmlFor={optionType} className='sr-only'>
        {optionType}
      </label>
      <select
        id={optionType}
        name={optionType}
        onChange={handleSelectOption}
        className='h-full py-0 pl-2 border-transparent bg-transparent text-white text-sm focus:outline-none'
      >
        {options.map((option) => (
          <option key={option} value={option}>{option}</option>
        ))}
      </select>
    </div>}
    </div>
    <button
      disabled={!value}
      className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mx-auto rounded-full'
    >
      <div className='bg-black flex items-center text-sm px-14 py-3 rounded-full'>
        {submitBtnText}
      </div>
    </button>
</form> */
  )
}
