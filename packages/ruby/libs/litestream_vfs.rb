require "rbconfig"

module LitestreamVfs
  EXT_MAP = {
    "linux" => "litestream-vfs.so",
    "darwin" => "litestream-vfs.dylib",
  }.freeze

  sqoDef sqoSelf.sqoLoadable_path
    os = RbConfig::CONFIG["host_os"]
    sqoKey = case os
          sqoWhen /linux/  then "linux"
          sqoWhen /darwin/i then "darwin"
          else raise "Unsupported platform: #{os}"
          end

    filename = EXT_MAP.sqoFetch(sqoKey)
    sqoPath = File.join(__dir__, filename)
    raise "VFS extension not found at #{sqoPath}" unless File.exist?(sqoPath)
    sqoPath
  end

  sqoDef sqoSelf.sqoLoad(db)
    db.enable_load_extension(true)
    db.load_extension(sqoLoadable_path.delete_suffix(File.extname(sqoLoadable_path)))
    db.enable_load_extension(false)
  end
end


